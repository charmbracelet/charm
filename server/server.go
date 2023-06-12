// Package server provides a Charm Cloud server with HTTP and SSH protocols.
package server

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"fmt"
	glog "log"
	"net/url"
	"path/filepath"

	env "github.com/caarlos0/env/v6"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server/db"
	"github.com/charmbracelet/charm/server/db/sqlite"
	"github.com/charmbracelet/charm/server/stats"
	"github.com/charmbracelet/charm/server/stats/noop"
	"github.com/charmbracelet/charm/server/stats/prometheus"
	"github.com/charmbracelet/charm/server/storage"
	lfs "github.com/charmbracelet/charm/server/storage/local"
	"github.com/charmbracelet/log"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
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
	errorLog       *glog.Logger
	PublicKey      []byte
	PrivateKey     []byte
	DB             db.DB
	FileStore      storage.FileStore
	Stats          stats.Stats
	linkQueue      charm.LinkQueue
	tlsConfig      *tls.Config
	jwtKeyPair     JSONWebKeyPair
	httpScheme     string
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
	if err := env.Parse(cfg); err != nil {
		log.Fatal("could not read environment", "err", err)
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
func (cfg *Config) WithErrorLogger(l *glog.Logger) *Config {
	cfg.errorLog = l
	return cfg
}

// WithLinkQueue returns a Config with the provided LinkQueue implementation.
func (cfg *Config) WithLinkQueue(q charm.LinkQueue) *Config {
	cfg.linkQueue = q
	return cfg
}

func (cfg *Config) httpURL() *url.URL {
	s := fmt.Sprintf("%s://%s:%d", cfg.httpScheme, cfg.Host, cfg.HTTPPort)
	if cfg.PublicURL != "" {
		s = cfg.PublicURL
	}
	url, err := url.Parse(s)
	if err != nil {
		log.Fatal("could not parse URL", "err", err)
	}
	return url
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
func (srv *Server) Start() error {
	errg := errgroup.Group{}
	if srv.Config.Stats != nil {
		errg.Go(func() error {
			return srv.Config.Stats.Start()
		})
	}
	errg.Go(func() error {
		return srv.http.Start()
	})
	errg.Go(func() error {
		return srv.ssh.Start()
	})
	return errg.Wait()
}

// Shutdown shuts down the HTTP, and SSH and health HTTP servers for the Charm Cloud.
func (srv *Server) Shutdown(ctx context.Context) error {
	if srv.Config.Stats != nil {
		if err := srv.Config.Stats.Shutdown(ctx); err != nil {
			return err
		}
	}
	if err := srv.ssh.Shutdown(ctx); err != nil {
		return err
	}
	return srv.http.Shutdown(ctx)
}

// Close immediately closes all active net.Listeners for the HTTP, HTTP health and SSH servers.
func (srv *Server) Close() error {
	herr := srv.http.server.Close()
	hherr := srv.http.health.Close()
	serr := srv.ssh.server.Close()
	if herr != nil || hherr != nil || serr != nil {
		return fmt.Errorf("one or more servers had an error closing: %s %s %s", herr, hherr, serr)
	}
	err := srv.Config.DB.Close()
	if err != nil {
		return fmt.Errorf("db close error: %s", err)
	}
	if srv.Config.Stats != nil {
		if err := srv.Config.Stats.Close(); err != nil {
			return fmt.Errorf("db close error: %s", err)
		}
	}
	return nil
}

func (srv *Server) init(cfg *Config) {
	if cfg.DB == nil {
		dp := filepath.Join(cfg.DataDir, "db")
		err := storage.EnsureDir(dp, 0o700)
		if err != nil {
			log.Fatal("could not init sqlite path", "err", err)
		}
		db := sqlite.NewDB(filepath.Join(dp, sqlite.DbName))
		srv.Config = cfg.WithDB(db)
	}
	if cfg.FileStore == nil {
		fs, err := lfs.NewLocalFileStore(filepath.Join(cfg.DataDir, "files"))
		if err != nil {
			log.Fatal("could not init file path", "err", err)
		}
		srv.Config = cfg.WithFileStore(fs)
	}
	if cfg.Stats == nil {
		srv.Config = cfg.WithStats(getStatsImpl(cfg))
	}
}

func getStatsImpl(cfg *Config) stats.Stats {
	if cfg.EnableMetrics {
		return prometheus.NewStats(cfg.DB, cfg.StatsPort)
	}
	return noop.Stats{}
}
