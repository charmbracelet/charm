// Package server provides a Charm Cloud server with HTTP and SSH protocols.
package server

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"fmt"
	"log"
	"path/filepath"

	"github.com/charmbracelet/charm/server/db"
	"github.com/charmbracelet/charm/server/db/sqlite"
	"github.com/charmbracelet/charm/server/stats"
	sls "github.com/charmbracelet/charm/server/stats/sqlite"
	"github.com/charmbracelet/charm/server/storage"
	lfs "github.com/charmbracelet/charm/server/storage/local"
	"github.com/meowgorithm/babyenv"
	gossh "golang.org/x/crypto/ssh"
)

// Config is the configuration for the Charm server.
type Config struct {
	Host        string `env:"CHARM_SERVER_HOST"`
	SSHPort     int    `env:"CHARM_SERVER_SSH_PORT" default:"35353"`
	HTTPPort    int    `env:"CHARM_SERVER_HTTP_PORT" default:"35354"`
	StatsPort   int    `env:"CHARM_SERVER_STATS_PORT" default:"35355"`
	HealthPort  int    `env:"CHARM_SERVER_HEALTH_PORT" default:"35356"`
	DataDir     string `env:"CHARM_SERVER_DATA_DIR" default:"./data"`
	TLSKeyFile  string `env:"CHARM_SERVER_TLS_KEY_FILE"`
	TLSCertFile string `env:"CHARM_SERVER_TLS_CERT_FILE"`
	errorLog    *log.Logger
	PublicKey   []byte
	PrivateKey  []byte
	DB          db.DB
	FileStore   storage.FileStore
	Stats       stats.Stats
	tlsConfig   *tls.Config
	jwtKeyPair  JSONWebKeyPair
	httpScheme  string
}

// Server contains the SSH and HTTP servers required to host the Charm Cloud.
type Server struct {
	Config *Config
	ssh    *SSHServer
	http   *HTTPServer
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	cfg := &Config{httpScheme: "http"}
	if err := babyenv.Parse(cfg); err != nil {
		log.Fatalf("could not read environment: %s", err)
	}

	return cfg
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

// WithTLSConfig returns a Config with the provided TLS configuration.
func (cfg *Config) WithTLSConfig(c *tls.Config) *Config {
	cfg.tlsConfig = c
	return cfg
}

// WithErrorLogger returns a Config with the provided error log for the server.
func (cfg *Config) WithErrorLogger(l *log.Logger) *Config {
	cfg.errorLog = l
	return cfg
}

func (cfg *Config) httpURL() string {
	return fmt.Sprintf("%s://%s:%d", cfg.httpScheme, cfg.Host, cfg.HTTPPort)
}

// NewServer returns a *Server with the specified Config.
func NewServer(cfg *Config) (*Server, error) {
	s := &Server{Config: cfg}
	s.init(cfg)

	pk, err := gossh.ParseRawPrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	cfg.jwtKeyPair = NewJSONWebKeyPair(pk.(*ed25519.PrivateKey))

	// Use HTTPS when TLS is configured.
	if cfg.tlsConfig != nil || (cfg.TLSCertFile != "" && cfg.TLSKeyFile != "") {
		cfg.httpScheme = "https"
	}

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

// Start starts the HTTP, SSH and health HTTP servers for the Charm Cloud.
func (srv *Server) Start() {
	go func() {
		srv.http.Start()
	}()
	srv.ssh.Start()
}

// Shutdown shuts down the HTTP, and SSH and health HTTP servers for the Charm Cloud.
func (srv *Server) Shutdown(ctx context.Context) error {
	if err := srv.ssh.Shutdown(ctx); err != nil {
		return err
	}
	return srv.http.Shutdown(ctx)
}

func (srv *Server) init(cfg *Config) {
	if cfg.DB == nil {
		dp := filepath.Join(cfg.DataDir, "db")
		err := storage.EnsureDir(dp, 0700)
		if err != nil {
			log.Fatalf("could not init sqlite path: %s", err)
		}
		db := sqlite.NewDB(dp)
		srv.Config = cfg.WithDB(db)
	}
	if cfg.FileStore == nil {
		fs, err := lfs.NewLocalFileStore(filepath.Join(cfg.DataDir, "files"))
		if err != nil {
			log.Fatalf("could not init file path: %s", err)
		}
		srv.Config = cfg.WithFileStore(fs)
	}
	if cfg.Stats == nil {
		sts, err := sls.NewStats(filepath.Join(cfg.DataDir, "stats"))
		if err != nil {
			log.Fatalf("could not init stats db: %s", err)
		}
		srv.Config = cfg.WithStats(sts)
	}
}
