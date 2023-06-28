// Package client manages authorization, identity and keys for a Charm Cloud
// user. It also offers low-level HTTP and SSH APIs for accessing the Charm
// Cloud server.
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	env "github.com/caarlos0/env/v6"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/keygen"
	jwt "github.com/golang-jwt/jwt/v4"
	homedir "github.com/mitchellh/go-homedir"
	gap "github.com/muesli/go-app-paths"
	"golang.org/x/crypto/ssh"
)

var nameValidator = regexp.MustCompile("^[a-zA-Z0-9]{1,50}$")

// Config contains the Charm client configuration.
type Config struct {
	Host        string `env:"CHARM_HOST" envDefault:"cloud.charm.sh"`
	SSHPort     int    `env:"CHARM_SSH_PORT" envDefault:"35353"`
	HTTPPort    int    `env:"CHARM_HTTP_PORT" envDefault:"35354"`
	Debug       bool   `env:"CHARM_DEBUG" envDefault:"false"`
	Logfile     string `env:"CHARM_LOGFILE" envDefault:""`
	KeyType     string `env:"CHARM_KEY_TYPE" envDefault:"ed25519"`
	DataDir     string `env:"CHARM_DATA_DIR" envDefault:""`
	IdentityKey string `env:"CHARM_IDENTITY_KEY" envDefault:""`
}

// Client is the Charm client.
type Client struct {
	Config               *Config
	auth                 *charm.Auth
	claims               *jwt.RegisteredClaims
	authLock             *sync.Mutex
	sshConfig            *ssh.ClientConfig
	httpScheme           string
	plainTextEncryptKeys []*charm.EncryptKey
	authKeyPaths         []string
	encryptKeyLock       *sync.Mutex
}

// ConfigFromEnv loads the configuration from the environment.
func ConfigFromEnv() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
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

	var sshKeys []string
	var err error
	if cfg.IdentityKey != "" {
		sshKeys = []string{cfg.IdentityKey}
	} else {
		sshKeys, err = cc.findAuthKeys(cfg.KeyType)
		if err != nil {
			return nil, err
		}
		if len(sshKeys) == 0 {
			dp, err := cc.DataPath()
			if err != nil {
				return nil, err
			}

			_, err = keygen.New(filepath.Join(dp, "charm_"+cfg.KeygenType().String()), keygen.WithKeyType(cfg.KeygenType()), keygen.WithWrite())
			if err != nil {
				return nil, err
			}
			sshKeys, err = cc.findAuthKeys(cfg.KeyType)
			if err != nil {
				return nil, err
			}
		}
	}

	var pkam ssh.AuthMethod
	for i := 0; i < len(sshKeys); i++ {
		signer, err := parseKey(sshKeys[i])
		if err != nil && i == len(sshKeys)-1 {
			return nil, charm.ErrMissingSSHAuth
		}
		if err := checkKeyAlgo(signer); err != nil && i == len(sshKeys)-1 {
			return nil, err
		}
		pkam = ssh.PublicKeys(signer)
	}
	cc.authKeyPaths = sshKeys

	cc.sshConfig = &ssh.ClientConfig{
		User:            "charm",
		Auth:            []ssh.AuthMethod{pkam},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // nolint
	}
	return cc, nil
}

// NewClientWithDefaults creates a new Charm client with default values.
func NewClientWithDefaults() (*Client, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	cc, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return cc, nil
}

// JWT returns a JSON web token for the user.
func (cc *Client) JWT(aud ...string) (string, error) {
	s, err := cc.sshSession()
	if err != nil {
		return "", err
	}
	defer s.Close() // nolint:errcheck
	jwt, err := s.Output(strings.Join(append([]string{"jwt"}, aud...), " "))
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
	defer s.Close() // nolint:errcheck
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
	defer s.Close() // nolint:errcheck
	keys, err := s.Output("keys")
	if err != nil {
		return "", err
	}
	return string(keys), nil
}

// AuthorizedKeysWithMetadata fetches keys linked to a user's account, with metadata.
func (cc *Client) AuthorizedKeysWithMetadata() (*charm.Keys, error) {
	s, err := cc.sshSession()
	if err != nil {
		return nil, err
	}
	defer s.Close() // nolint:errcheck

	b, err := s.Output("api-keys")
	if err != nil {
		return nil, err
	}

	var k charm.Keys
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
	defer s.Close() // nolint:errcheck
	k := charm.PublicKey{Key: key}
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

// KeygenType returns the keygen key type.
func (cfg *Config) KeygenType() keygen.KeyType {
	kt := strings.ToLower(cfg.KeyType)
	switch kt {
	case "ed25519":
		return keygen.Ed25519
	case "rsa":
		return keygen.RSA
	default:
		return keygen.Ed25519
	}
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
	if (*u == charm.User{}) {
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

// DataPath return the directory a Charm user's data is stored. It will default
// to XDG-HOME/$CHARM_HOST.
func (cc *Client) DataPath() (string, error) {
	if cc.Config.DataDir != "" {
		return filepath.Join(cc.Config.DataDir, cc.Config.Host), nil
	}
	scope := gap.NewScope(gap.User, filepath.Join("charm", cc.Config.Host))
	dataPath, err := scope.DataPath("")
	if err != nil {
		return "", err
	}
	return dataPath, nil
}

// FindAuthKeys looks in a user's XDG charm-dir for possible auth keys.
// If no keys are found we return an empty slice.
func (cc *Client) findAuthKeys(keyType string) (pathsToKeys []string, err error) {
	keyPath, err := cc.DataPath()
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
		if filepath.Base(f) == fmt.Sprintf("charm_%s", keyType) {
			found = append(found, f)
		}
	}

	return found, nil
}

func checkKeyAlgo(signer ssh.Signer) error {
	ka := signer.PublicKey().Type()
	for _, a := range []string{"ssh-rsa", "ssh-ed25519"} {
		if a == ka {
			return nil
		}
	}
	return fmt.Errorf("Sorry, we don't support %s keys yet. Supported types are rsa and ed25519", algo(ka))
}

func parseKey(kp string) (ssh.Signer, error) {
	keyPath, err := homedir.Expand(kp)
	if err != nil {
		return nil, err
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return signer, nil
}
