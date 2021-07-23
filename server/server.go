// Package server provides a Charm Cloud server with HTTP and SSH protocols.
package server

import (
	"fmt"
	"log"

	"github.com/charmbracelet/charm/server/db"
	"github.com/charmbracelet/charm/server/db/sqlite"
	"github.com/charmbracelet/charm/server/stats"
	sls "github.com/charmbracelet/charm/server/stats/sqlite"
	"github.com/charmbracelet/charm/server/storage"
	lfs "github.com/charmbracelet/charm/server/storage/local"
	"github.com/meowgorithm/babyenv"
)

// Config is the configuration for the Charm server.
type Config struct {
	Host        string `env:"CHARM_SERVER_HOST" default:"localhost"`
	SSHPort     int    `env:"CHARM_SERVER_SSH_PORT" default:"35353"`
	HTTPPort    int    `env:"CHARM_SERVER_HTTP_PORT" default:"35354"`
	HTTPScheme  string `env:"CHARM_SERVER_HTTP_SCHEME" default:"http"`
	StatsPort   int    `env:"CHARM_SERVER_STATS_PORT" default:"35355"`
	HealthPort  string `env:"CHARM_SERVER_HEALTH_PORT" default:"35356"`
	DataDir     string `env:"CHARM_SERVER_DATA_DIR" default:"./data"`
	TLSKey      []byte `env:"CHARM_SERVER_TLS_KEY" default:""`
	TLSCert     []byte `env:"CHARM_SERVER_TLS_CERT" default:""`
	TLSKeyFile  string `env:"CHARM_SERVER_TLS_KEY_FILE" default:""`
	TLSCertFile string `env:"CHARM_SERVER_TLS_CERT_FILE" default:""`
	PublicKey   []byte
	PrivateKey  []byte
	DB          db.DB
	FileStore   storage.FileStore
	Stats       stats.Stats
}

// Server contains the SSH and HTTP servers required to host the Charm Cloud.
type Server struct {
	Config        *Config
	ssh           *SSHServer
	http          *HTTPServer
	publicKey     []byte
	privateKeyPEM []byte
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	cfg := &Config{}
	err := babyenv.Parse(cfg)
	if err != nil {
		log.Fatalf("could not read environment: %s", err)
	}
	dp := fmt.Sprintf("%s/db", cfg.DataDir)
	err = storage.EnsureDir(dp, 0700)
	if err != nil {
		log.Fatalf("could not init sqlite path: %s", err)
	}
	db := sqlite.NewDB(dp)
	fs, err := lfs.NewLocalFileStore(fmt.Sprintf("%s/files", cfg.DataDir))
	if err != nil {
		log.Fatalf("could not init file path: %s", err)
	}
	sts, err := sls.NewStats(fmt.Sprintf("%s/stats", cfg.DataDir))
	if err != nil {
		log.Fatalf("could not init stats db: %s", err)
	}
	return cfg.WithDB(db).WithFileStore(fs).WithStats(sts)
}

// WithDB returns a Config with the provided DB interface implementation.
func (cfg *Config) WithDB(db db.DB) *Config {
	cfg.DB = db
	return cfg
}

// WithFileStore returns a Config with the provided FileStore implementation.
func (cfg *Config) WithFileStore(fs storage.FileStore) *Config {
	cfg.FileStore = fs
	return cfg
}

// WithStats returns a Config with the provided Stats implementation.
func (cfg *Config) WithStats(s stats.Stats) *Config {
	cfg.Stats = s
	return cfg
}

// WithKeys returns a Config with the provided public and private keys for the
// SSH server and JWT signing.
func (cfg *Config) WithKeys(publicKey []byte, privateKey []byte) *Config {
	cfg.PublicKey = publicKey
	cfg.PrivateKey = privateKey
	return cfg
}

// NewServer returns a *Server with the specified Config.
func NewServer(cfg *Config) (*Server, error) {
	s := &Server{Config: cfg}
	ss, err := NewSSHServer(cfg)
	if err != nil {
		return nil, err
	}
	s.ssh = ss
	hs, err := NewHTTPServer(cfg)
	if err != nil {
		return nil, err
	}
	s.http = hs
	return s, nil
}

// Start starts the HTTP, SSH and stats HTTP servers for the Charm Cloud.
func (srv *Server) Start() {
	go func() {
		srv.http.Start()
	}()
	srv.ssh.Start()
}
