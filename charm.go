package charm

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/meowgorithm/babyenv"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var nameValidator = regexp.MustCompile("^[a-zA-Z0-9]{1,50}$")

// ErrMissingSSHAuth is used when the user is missing SSH credentials
var ErrMissingSSHAuth = errors.New("missing ssh auth")

// ErrNameTaken is used when a user attempts to set a username and that
// username is already taken
var ErrNameTaken = errors.New("name already taken")

// ErrNameInvalid is used when a username is invalid
var ErrNameInvalid = errors.New("invalid name")

// ErrCouldNotUnlinkKey is used when a key can't be deleted
var ErrCouldNotUnlinkKey = errors.New("could not unlink key")

// Config contains the Charm client configuration
type Config struct {
	IDHost      string `env:"CHARM_ID_HOST" default:"id.charm.sh"`
	IDPort      int    `env:"CHARM_ID_PORT" default:"22"`
	BioHost     string `env:"CHARM_BIO_HOST" default:"https://bio.charm.sh"`
	BioPort     int    `env:"CHARM_BIO_PORT" default:"443"`
	GlowHost    string `env:"CHARM_GLOW_HOST" default:"https://glow.charm.sh"`
	GlowPort    int    `env:"CHARM_GLOW_PORT" default:"443"`
	JWTKey      string `env:"CHARM_JWT_KEY" default:""`
	UseSSHAgent bool   `env:"CHARM_USE_SSH_AGENT" default:"true"`
	SSHKeyPath  string `env:"CHARM_SSH_KEY_PATH" default:"~/.ssh/id_rsa"`
	Debug       bool   `env:"CHARM_DEBUG" default:"false"`
	Logfile     string `env:"CHARM_LOGFILE" default:""`
	ForceKey    bool
}

// Client is the Charm client
type Client struct {
	auth               *Auth
	config             *Config
	sshConfig          *ssh.ClientConfig
	jwtPublicKey       *rsa.PublicKey
	authLock           *sync.Mutex
	initialSession     *ssh.Session
	initialSessionOnce *sync.Once
}

// User represents a Charm user
type User struct {
	CharmID   string     `json:"charm_id"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Bio       string     `json:"bio"`
	CreatedAt *time.Time `json:"created_at"`
}

// Keys is a server response returned when the user queries for the keys linked
// to her account.
type Keys struct {
	ActiveKey int   `json:"active_key"`
	Keys      []Key `json:"keys"`
}

// ConfigFromEnv loads the configuration from the environment
func ConfigFromEnv() (*Config, error) {
	var cfg Config
	if err := babyenv.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewClient creates a new Charm client
func NewClient(cfg *Config) (*Client, error) {
	cc := &Client{
		config:             cfg,
		auth:               &Auth{},
		authLock:           &sync.Mutex{},
		initialSessionOnce: &sync.Once{},
	}
	err := cc.setJWTKey()
	if err != nil {
		return nil, err
	}

	var sshKeys []string

	if cfg.ForceKey && cfg.SSHKeyPath != "" {

		// User wants to use a specific key
		ext := filepath.Ext(cfg.SSHKeyPath)
		if ext == ".pub" {
			sshKeys = []string{
				cfg.SSHKeyPath,
				strings.TrimSuffix(cfg.SSHKeyPath, ext),
			}
		} else {
			sshKeys = []string{
				cfg.SSHKeyPath + ".pub",
				cfg.SSHKeyPath,
			}
		}

	} else {

		// Try and use SSH agent for auth
		am, err := agentAuthMethod()
		if err == nil {
			cc.sshConfig = &ssh.ClientConfig{
				User:            "charm",
				Auth:            []ssh.AuthMethod{am},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			}

			// Dial session here as agent may still not work
			c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cc.config.IDHost, cc.config.IDPort), cc.sshConfig)
			if err == nil {
				// Successful dial; let's try and finish up here

				s, err := c.NewSession()
				if err == nil {
					// It worked. Cache dialed session and use SSH agent for auth!
					cc.initialSession = s
					return cc, nil
				}
			}
		}

		// If we're still here it means SSH agent either failed or isn't setup, so
		// now we look for default SSH keys.
		sshKeys, err = findSSHKeys()
		if err != nil {
			return nil, err
		}
		if len(sshKeys) == 0 { // We didn't find any keys; give up
			return nil, ErrMissingSSHAuth
		}

	}

	// Try and use the keys we found
	var pkam ssh.AuthMethod
	for i := 0; i < len(sshKeys); i++ {
		pkam, err = publicKeyAuthMethod(sshKeys[i])
		if err != nil && i == len(sshKeys)-1 {
			return nil, ErrMissingSSHAuth
		}
	}

	cc.sshConfig = &ssh.ClientConfig{
		User:            "charm",
		Auth:            []ssh.AuthMethod{pkam},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return cc, nil
}

// JWT returns a JSON web token for the user
func (cc *Client) JWT() (string, error) {
	s, err := cc.sshSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	jwt, err := s.Output("jwt")
	if err != nil {
		return "", err
	}
	return string(jwt), nil
}

// ID returns the user's ID
func (cc *Client) ID() (string, error) {
	s, err := cc.sshSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	id, err := s.Output("id")
	if err != nil {
		return "", err
	}
	return string(id), nil
}

// AuthorizedKeys returns the keys linked to a user's account
func (cc *Client) AuthorizedKeys() (string, error) {
	s, err := cc.sshSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	keys, err := s.Output("keys")
	if err != nil {
		return "", err
	}
	return string(keys), nil
}

// AuthorizedKeys fetches keys linked to a user's account, with metadata
func (cc *Client) AuthorizedKeysWithMetadata() (*Keys, error) {
	s, err := cc.sshSession()
	if err != nil {
		return nil, err
	}
	defer s.Close()
	b, err := s.Output("api-keys")
	if err != nil {
		return nil, err
	}
	var k Keys
	err = json.Unmarshal(b, &k)
	return &k, err
}

// UnlinkAuthorizedKey removes an authorized key from the user's Charm account
func (cc *Client) UnlinkAuthorizedKey(key string) error {
	s, err := cc.sshSession()
	if err != nil {
		return err
	}
	defer s.Close()
	k := Key{Key: key}
	in, err := s.StdinPipe()
	if err != nil {
		return err
	}
	if err := json.NewEncoder(in).Encode(k); err != nil {
		return err
	}
	j, err := json.Marshal(&k)
	if err != nil {
		return err
	}
	b, err := s.Output(fmt.Sprintf("api-unlink %s", string(j)))
	if err != nil {
		return err
	}
	if len(b) != 0 {
		return ErrCouldNotUnlinkKey
	}
	return nil
}

// Link joins in on a linking session initiated by LinkGen
func (cc *Client) Link(lh LinkHandler, code string) error {
	s, err := cc.sshSession()
	if err != nil {
		return err
	}
	defer s.Close()
	out, err := s.StdoutPipe()
	if err != nil {
		return err
	}

	err = s.Start(fmt.Sprintf("api-link %s", code))
	if err != nil {
		return err
	}
	var lr Link
	dec := json.NewDecoder(out)
	err = dec.Decode(&lr)
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr) {
		return nil
	}

	var lr2 Link
	err = dec.Decode(&lr2)
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr2) {
		return nil
	}

	var lr3 Link
	err = dec.Decode(&lr3)
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr3) {
		return nil
	}
	return nil
}

// LinkGen initiates a linking session
func (cc *Client) LinkGen(lh LinkHandler) error {
	s, err := cc.sshSession()
	if err != nil {
		return err
	}
	defer s.Close()
	out, err := s.StdoutPipe()
	if err != nil {
		return err
	}
	in, err := s.StdinPipe()
	if err != nil {
		return err
	}

	err = s.Start("api-link")
	if err != nil {
		return err
	}

	// initialize link request on server
	var lr Link
	dec := json.NewDecoder(out)
	err = dec.Decode(&lr)
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr) {
		return nil
	}

	// waiting for link request, do we want to approve it?
	var lr2 Link
	err = dec.Decode(&lr2)
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr2) {
		return nil
	}

	// send approval response
	var lm LinkerMessage
	enc := json.NewEncoder(in)
	if lh.Request(&lr2) {
		lm = LinkerMessage{"yes"}
	} else {
		lm = LinkerMessage{"no"}
	}
	err = enc.Encode(lm)
	if err != nil {
		return err
	}
	if lm.Message == "no" {
		return nil
	}

	// get server response
	var lr3 Link
	err = dec.Decode(&lr3)
	if err != nil {
		return err
	}
	checkLinkStatus(lh, &lr3)
	return nil
}

// SetName sets the account's username
func (cc *Client) SetName(name string) (*User, error) {
	if !ValidateName(name) {
		return nil, ErrNameInvalid
	}
	u := &User{}
	u.Name = name
	client := &http.Client{}
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(u)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s:%d/v1/bio", cc.config.BioHost, cc.config.BioPort), buf)
	if err != nil {
		return nil, err
	}
	jwt, err := cc.JWT()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusConflict {
		return nil, ErrNameTaken
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Bio returns the user's profile
func (cc *Client) Bio() (*User, error) {
	u := &User{}
	client := &http.Client{}
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(u)
	if err != nil {
		return nil, err
	}
	id, err := cc.ID()
	if err != nil {
		return nil, err
	}
	jwt, err := cc.JWT()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s:%d/v1/id/%s", cc.config.BioHost, cc.config.BioPort, id), buf)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusConflict {
		return nil, ErrNameTaken
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ValidateName validates a given name
func ValidateName(name string) bool {
	return nameValidator.MatchString(name)
}

func (cc *Client) Auth() (*Auth, error) {
	cc.authLock.Lock()
	defer cc.authLock.Unlock()
	if cc.auth.claims == nil || cc.auth.claims.Valid() != nil {
		auth := &Auth{}
		s, err := cc.sshSession()
		if err != nil {
			return nil, err
		}
		defer s.Close()
		b, err := s.Output("api-auth")
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, auth)
		if err != nil {
			return nil, err
		}
		token, err := jwt.ParseWithClaims(auth.JWT, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
			return cc.jwtPublicKey, nil
		})
		if err != nil {
			return nil, err
		}
		auth.claims = token.Claims.(*jwt.StandardClaims)
		cc.auth = auth
		if err != nil {
			return nil, err
		}
	}
	return cc.auth, nil
}

func (cc *Client) sshSession() (*ssh.Session, error) {
	var s *ssh.Session

	// On first run we may have already dialed a session to test ssh agent
	cc.initialSessionOnce.Do(func() {
		s = cc.initialSession
	})
	if s != nil {
		return s, nil
	}

	c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cc.config.IDHost, cc.config.IDPort), cc.sshConfig)
	if err != nil {
		return nil, err
	}
	s, err = c.NewSession()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}
	return true
}

func publicKeyAuthMethod(kp string) (ssh.AuthMethod, error) {
	keyPath, err := homedir.Expand(kp)
	if err != nil {
		return nil, err
	}
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

func agentAuthMethod() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		// fmt.Println("No SSH_AUTH_SOCK set, not using ssh-agent")
		return nil, fmt.Errorf("Missing socket env var")
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		// fmt.Printf("SSH agent dial error: %s\n", err)
		return nil, err
	}
	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

// findSSHKeys looks in a user's ~/.ssh dir for possible SSH keys. If no keys
// are found we return an empty slice.
func findSSHKeys() (pathsToKeys []string, err error) {
	path, err := homedir.Expand("~/.ssh")
	if err != nil {
		return nil, err
	}

	m, err := filepath.Glob(filepath.Join(path, "id_*"))
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return nil, nil
	}

	var found []string
	for _, f := range m {
		switch filepath.Base(f) {
		case "id_dsa":
			fallthrough
		case "id_rsa":
			fallthrough
		case "id_ecdsa":
			fallthrough
		case "id_ed25519":
			found = append(found, f)
		}
	}

	return found, nil
}
