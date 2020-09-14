package charm

import (
	"crypto/rsa"
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
	"time"

	"github.com/meowgorithm/babyenv"
	"github.com/mitchellh/go-homedir"
	gap "github.com/muesli/go-app-paths"
	"github.com/muesli/sasquatch"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var nameValidator = regexp.MustCompile("^[a-zA-Z0-9]{1,50}$")

// ErrMissingSSHAuth is used when the user is missing SSH credentials.
var ErrMissingSSHAuth = errors.New("missing ssh auth")

// ErrNameTaken is used when a user attempts to set a username and that
// username is already taken.
var ErrNameTaken = errors.New("name already taken")

// ErrNameInvalid is used when a username is invalid.
var ErrNameInvalid = errors.New("invalid name")

// ErrCouldNotUnlinkKey is used when a key can't be deleted.
var ErrCouldNotUnlinkKey = errors.New("could not unlink key")

// ErrAuthFailed indicates an authentication failure. The underlying error is
// wrapped.
type ErrAuthFailed struct {
	Err error
}

// Error returns the boxed error string.
func (e ErrAuthFailed) Error() string { return e.Err.Error() }

// Unwrap returns the boxed error.
func (e ErrAuthFailed) Unwrap() error { return e.Err }

// Config contains the Charm client configuration.
type Config struct {
	IDHost   string `env:"CHARM_ID_HOST" default:"id.charm.sh"`
	IDPort   int    `env:"CHARM_ID_PORT" default:"22"`
	BioHost  string `env:"CHARM_BIO_HOST" default:"https://bio.charm.sh"`
	BioPort  int    `env:"CHARM_BIO_PORT" default:"443"`
	GlowHost string `env:"CHARM_GLOW_HOST" default:"https://glow.charm.sh"`
	GlowPort int    `env:"CHARM_GLOW_PORT" default:"443"`
	JWTKey   string `env:"CHARM_JWT_KEY" default:""`
	Debug    bool   `env:"CHARM_DEBUG" default:"false"`
	Logfile  string `env:"CHARM_LOGFILE" default:""`
}

// Client is the Charm client.
type Client struct {
	config               *Config
	auth                 *Auth
	authLock             *sync.Mutex
	sshConfig            *ssh.ClientConfig
	jwtPublicKey         *rsa.PublicKey
	plainTextEncryptKeys []*EncryptKey
	authKeyPaths         []string
	encryptKeyLock       *sync.Mutex
}

// User represents a Charm user.
type User struct {
	CharmID   string    `json:"charm_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
}

// Keys is a server response returned when the user queries for the keys linked
// to her account.
type Keys struct {
	ActiveKey int   `json:"active_key"`
	Keys      []Key `json:"keys"`
}

// DataPath returns the Charm data path for the current user. This is where
// Charm keys are stored.
func DataPath() (string, error) {
	scope := gap.NewScope(gap.User, "charm")
	dataPath, err := scope.DataPath("")
	if err != nil {
		return "", err
	}
	return dataPath, nil
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
		config:         cfg,
		auth:           &Auth{},
		authLock:       &sync.Mutex{},
		encryptKeyLock: &sync.Mutex{},
	}
	jk, err := jwtKey(cfg.JWTKey)
	if err != nil {
		return nil, err
	}
	cc.jwtPublicKey = jk

	sshKeys, err := findAuthKeys()
	if err != nil {
		return nil, err
	}
	if len(sshKeys) == 0 { // We didn't find any keys; give up
		return nil, ErrMissingSSHAuth
	}

	// Try and use the keys we found
	var pkam ssh.AuthMethod
	for i := 0; i < len(sshKeys); i++ {
		pkam, err = publicKeyAuthMethod(sshKeys[i])
		if err != nil && i == len(sshKeys)-1 {
			return nil, ErrMissingSSHAuth
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
		return ErrCouldNotUnlinkKey
	}
	return nil
}

// SetName sets the account's username.
func (cc *Client) SetName(name string) (*User, error) {
	if !ValidateName(name) {
		return nil, ErrNameInvalid
	}
	u := &User{}
	u.Name = name
	err := cc.AuthedRequest("POST", cc.config.BioHost, cc.config.BioPort, "/v1/bio", u, u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Bio returns the user's profile.
func (cc *Client) Bio() (*User, error) {
	u := &User{}
	id, err := cc.ID()
	if err != nil {
		return nil, err
	}
	err = cc.AuthedRequest("GET", cc.config.BioHost, cc.config.BioPort, fmt.Sprintf("/v1/id/%s", id), u, u)
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
	c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cc.config.IDHost, cc.config.IDPort), cc.sshConfig)
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

// findCharmKeys looks in a user's XDG charm-dir for possible auth keys.
// If no keys are found we return an empty slice.
func findAuthKeys() (pathsToKeys []string, err error) {
	keyPath, err := DataPath()
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

func findSSHSigners() []ssh.Signer {
	var r []ssh.Signer

	// from agent
	signers, err := sasquatch.SSHAgentSigners()
	if err == nil {
		r = append(r, signers...)
	}

	files, _ := findSSHKeys()
	for _, file := range files {
		k, err := parsePrivateKey(file)
		if err != nil {
			continue
		}

		signer, err := ssh.NewSignerFromKey(k)
		if err != nil {
			continue
		}

		r = append(r, signer)
	}

	return r
}
