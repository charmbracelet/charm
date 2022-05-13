package config

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"

	"github.com/caarlos0/env/v6"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server/db"
	"github.com/charmbracelet/charm/server/jwt"
	"github.com/charmbracelet/charm/server/stats"
	"github.com/charmbracelet/charm/server/storage"
)

var (
	// Version is the version of the Charm Cloud server. This is set at build
	// time by main.go.
	Version = ""
)

// Config is the configuration for the Charm server.
type Config struct {
	BindAddr       string `env:"CHARM_SERVER_BIND_ADDRESS" envDefault:""`
	Host           string `env:"CHARM_SERVER_HOST" envDefault:"localhost"`
	SSHPort        int    `env:"CHARM_SERVER_SSH_PORT" envDefault:"35353"`
	HTTPPort       int    `env:"CHARM_SERVER_HTTP_PORT" envDefault:"35354"`
	StatsPort      int    `env:"CHARM_SERVER_STATS_PORT" envDefault:"35355"`
	HealthPort     int    `env:"CHARM_SERVER_HEALTH_PORT" envDefault:"35356"`
	DataDir        string `env:"CHARM_SERVER_DATA_DIR" envDefault:"data"`
	UseTLS         bool   `env:"CHARM_SERVER_USE_TLS" envDefault:"false"`
	TLSKeyFile     string `env:"CHARM_SERVER_TLS_KEY_FILE"`
	TLSCertFile    string `env:"CHARM_SERVER_TLS_CERT_FILE"`
	PublicURL      string `env:"CHARM_SERVER_PUBLIC_URL"`
	EnableMetrics  bool   `env:"CHARM_SERVER_ENABLE_METRICS" envDefault:"false"`
	UserMaxStorage int64  `env:"CHARM_SERVER_USER_MAX_STORAGE" envDefault:"0"`
	ErrorLog       *log.Logger
	PublicKey      []byte
	PrivateKey     []byte
	DB             db.DB
	FileStore      storage.FileStore
	Stats          stats.Stats
	LinkQueue      charm.LinkQueue
	TLSConfig      *tls.Config
	JWTKeyPair     jwt.JWTKeyPair
}

// DefaultConfig returns a Config with the values populated with the defaults
// or specified environment variables.
func DefaultConfig() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
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
	cfg.TLSConfig = c
	return cfg
}

// WithErrorLogger returns a Config with the provided error log for the server.
func (cfg *Config) WithErrorLogger(l *log.Logger) *Config {
	cfg.ErrorLog = l
	return cfg
}

// WithLinkQueue returns a Config with the provided LinkQueue implementation.
func (cfg *Config) WithLinkQueue(q charm.LinkQueue) *Config {
	cfg.LinkQueue = q
	return cfg
}

// WithJWTKeyPair returns a Config with the provided JWT key pair.
func (cfg *Config) WithJWTKeyPair(k jwt.JWTKeyPair) *Config {
	cfg.JWTKeyPair = k
	return cfg
}

// HttpUrl returns the URL for the HTTP server.
func (cfg *Config) HTTPURL() *url.URL {
	s := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.HTTPPort)
	if cfg.PublicURL != "" {
		s = cfg.PublicURL
	}
	url, err := url.Parse(s)
	if err != nil {
		log.Fatalf("could not parse URL: %s", err)
	}
	return url
}
