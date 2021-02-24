package server

import (
	"fmt"
	"log"

	"github.com/charmbracelet/charm/server/sqlite"
	"github.com/meowgorithm/babyenv"
)

type Config struct {
	Host       string `env:"CHARM_HOST" default:"localhost"`
	SSHPort    int    `env:"CHARM_SSH_PORT" default:"35353"`
	HTTPPort   int    `env:"CHARM_HTTP_PORT" default:"35354"`
	StatsPort  int    `env:"CHARM_STATS_PORT" default:"35355"`
	HealthPort string `env:"CHARM_HEALTH_PORT" default:"35356"`
	DataDir    string `env:"CHARM_DATA_DIR" default:"./data"`
	PublicKey  []byte
	PrivateKey []byte
	DB         DB
	FileStore  FileStore
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
	err := babyenv.Parse(&cfg)
	if err != nil {
		log.Fatalf("could not read environment: %s", err)
	}
	dp := fmt.Sprintf("%s/db", cfg.DataDir)
	err = EnsureDir(dp)
	if err != nil {
		log.Fatalf("could not init sqlite path: %s", err)
	}
	db := sqlite.NewDB(dp)
	fs, err := NewLocalFileStore(fmt.Sprintf("%s/files", cfg.DataDir))
	if err != nil {
		log.Fatalf("could not init file path: %s", err)
	}
	return cfg.WithDB(db).WithFileStore(fs).WithStats(NewPrometheusStats(db, cfg.StatsPort))
}

func (cfg Config) WithDB(db DB) Config {
	cfg.DB = db
	return cfg
}

func (cfg Config) WithFileStore(fs FileStore) Config {
	cfg.FileStore = fs
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
