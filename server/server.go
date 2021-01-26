package server

import (
	"fmt"

	"github.com/charmbracelet/charm/server/sqlite"
	"github.com/meowgorithm/babyenv"
)

type Config struct {
	Host       string `env:"CHARM_HOST" default:"localhost"`
	SSHPort    int    `env:"CHARM_SSH_PORT" default:"35353"`
	HTTPPort   int    `env:"CHARM_HTTP_PORT" default:"35354"`
	StatsPort  int    `env:"CHARM_STATS_PORT" default:"35355"`
	HealthPort string `env:"CHARM_HEALTH_PORT" default:"35356"`
	PublicKey  []byte
	PrivateKey []byte
	Storage    Storage
	Stats      PrometheusStats
}

type Server struct {
	Config        Config
	ssh           *SSHServer
	http          *HTTPServer
	stats         PrometheusStats
	publicKey     []byte
	privateKeyPEM []byte
}

func DefaultConfig() Config {
	var cfg Config
	if err := babyenv.Parse(&cfg); err != nil {
		panic(fmt.Sprintf("could not read environment: %s", err))
	}
	db := sqlite.NewDB("./")
	return cfg.WithStorage(db).WithStats(NewPrometheusStats(db, cfg.StatsPort))
}

func (cfg Config) WithStorage(s Storage) Config {
	cfg.Storage = s
	return cfg
}

func (cfg Config) WithStats(ps PrometheusStats) Config {
	// TODO: make stats an interface
	cfg.Stats = ps
	return cfg
}

func (cfg Config) WithKeys(publicKey []byte, privateKey []byte) Config {
	cfg.PublicKey = publicKey
	cfg.PrivateKey = privateKey
	return cfg
}

func NewServer(cfg Config) *Server {
	s := &Server{Config: cfg}
	s.ssh = NewSSHServer(cfg)
	s.http = NewHTTPServer(cfg)
	s.stats = cfg.Stats
	return s
}

func (srv *Server) Start() {
	go func() {
		srv.stats.Start()
	}()
	go func() {
		srv.http.Start()
	}()
	srv.ssh.Start()
}
