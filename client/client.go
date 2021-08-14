// Package client manages authorization, identity and keys for a Charm Cloud
// user. It also offers low-level HTTP and SSH APIs for accessing the Charm
// Cloud server.
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/charmbracelet/charm/keygen"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/dgrijalva/jwt-go"
	"github.com/meowgorithm/babyenv"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var nameValidator = regexp.MustCompile("^[a-zA-Z0-9]{1,50}$")

// Config contains the Charm client configuration.
type Config struct {
	Host     string `env:"CHARM_HOST" default:"beta.charm.sh"`
	SSHPort  int    `env:"CHARM_SSH_PORT" default:"35353"`
	HTTPPort int    `env:"CHARM_HTTP_PORT" default:"35354"`
	Debug    bool   `env:"CHARM_DEBUG" default:"false"`
	Logfile  string `env:"CHARM_LOGFILE" default:""`
}

// Client is the Charm client.
type Client struct {
	Config               *Config
	auth                 *charm.Auth
	claims               *jwt.StandardClaims
	authLock             *sync.Mutex
	sshConfig            *ssh.ClientConfig
	httpScheme           string
	plainTextEncryptKeys []*charm.EncryptKey
	authKeyPaths         []string
	encryptKeyLock       *sync.Mutex
}

// Keys is a server response returned when the user queries for the keys linked
// to her account.
type Keys struct {
	ActiveKey int   `json:"active_key"`
	Keys      []Key `json:"keys"`
}

// ConfigFromEnv loads the configuration from the environment.
func ConfigFromEnv() (*Config, error) {
	var cfg Config
	if err := babyenv.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewClient creates a new Charm client.
func NewClient(cfg *Config) (*Client, error) {
	cc := &Client{
		Config:         cfg,
		auth:           &charm.Auth{},
		authLock:       &sync.Mutex{},
		encryptKeyLock: &sync.Mutex{},
	}
	sshKeys, err := FindAuthKeys(cfg.Host)
	if err != nil {
		return nil, err
	}
	if len(sshKeys) == 0 { // We didn't find any keys; give up
		return nil, charm.ErrMissingSSHAuth
	}

	// Try and use the keys we found
	var pkam ssh.AuthMethod
	for i := 0; i < len(sshKeys); i++ {
		pkam, err = publicKeyAuthMethod(sshKeys[i])
		if err != nil && i == len(sshKeys)-1 {
			return nil, charm.ErrMissingSSHAuth
		}
	}
	cc.authKeyPaths = sshKeys

	cc.sshConfig = &ssh.ClientConfig{
		User:            "charm",
		Auth:            []ssh.AuthMethod{pkam},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return cc, nil
}

func NewClientWithDefaults() (*Client, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	cc, err := NewClient(cfg)
	if err == charm.ErrMissingSSHAuth {
		dp, err := DataPath(cfg.Host)
		if err != nil {
			return nil, err
		}
		_, err = keygen.NewSSHKeyPair(dp, "charm", []byte(""), "rsa")
		if err != nil {
			return nil, err
		}
		return NewClient(cfg)
	} else if err != nil {
		return nil, err
	}
	return cc, nil
}

// JWT returns a JSON web token for the user.
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

// ID returns the user's ID.
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

// AuthorizedKeys returns the keys linked to a user's account.
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

// AuthorizedKeysWithMetadata fetches keys linked to a user's account, with metadata.
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

// AuthKeyPaths returns the full file path of the Charm auth SSH keys.
func (cc *Client) AuthKeyPaths() []string {
	return cc.authKeyPaths
}

// UnlinkAuthorizedKey removes an authorized key from the user's Charm account.
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
		return charm.ErrCouldNotUnlinkKey
	}
	return nil
}

// SetName sets the account's username.
func (cc *Client) SetName(name string) (*charm.User, error) {
	if !ValidateName(name) {
		return nil, charm.ErrNameInvalid
	}
	u := &charm.User{}
	u.Name = name
	err := cc.AuthedJSONRequest("POST", "/v1/bio", u, u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Bio returns the user's profile.
func (cc *Client) Bio() (*charm.User, error) {
	u := &charm.User{}
	id, err := cc.ID()
	if err != nil {
		return nil, err
	}
	err = cc.AuthedJSONRequest("GET", fmt.Sprintf("/v1/id/%s", id), u, u)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("no user data received")
	}
	return u, nil
}

// ValidateName validates a given name.
func ValidateName(name string) bool {
	return nameValidator.MatchString(name)
}

func (cc *Client) sshSession() (*ssh.Session, error) {
	cfg := cc.Config
	c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.SSHPort), cc.sshConfig)
	if err != nil {
		return nil, err
	}
	s, err := c.NewSession()
	if err != nil {
		return nil, err
	}
	return s, nil
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

// FindAuthKeys looks in a user's XDG charm-dir for possible auth keys.
// If no keys are found we return an empty slice.
func FindAuthKeys(host string) (pathsToKeys []string, err error) {
	keyPath, err := DataPath(host)
	if err != nil {
		return nil, err
	}
	m, err := filepath.Glob(filepath.Join(keyPath, "charm_*"))
	if err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return nil, nil
	}

	var found []string
	for _, f := range m {
		switch filepath.Base(f) {
		case "charm_rsa":
			fallthrough
		case "charm_ecdsa":
			fallthrough
		case "charm_ed25519":
			found = append(found, f)
		}
	}

	return found, nil
}

const privateKeySizeLimit = 1 << 24 // 16 MiB

func parsePrivateKey(file string) (interface{}, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	contents, err := ioutil.ReadAll(io.LimitReader(f, privateKeySizeLimit))
	if err != nil {
		return nil, err
	}
	if len(contents) == privateKeySizeLimit {
		return nil, fmt.Errorf("key size exceeded limit")
	}

	return ssh.ParseRawPrivateKey(contents)
}
