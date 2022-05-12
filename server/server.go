// Package server provides a Charm Cloud server with HTTP and SSH protocols.
package server

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"path/filepath"

	"github.com/charmbracelet/charm/server/config"
	"github.com/charmbracelet/charm/server/db/sqlite"
	"github.com/charmbracelet/charm/server/jwt"
	"github.com/charmbracelet/charm/server/stats"
	"github.com/charmbracelet/charm/server/stats/noop"
	"github.com/charmbracelet/charm/server/stats/prometheus"
	"github.com/charmbracelet/charm/server/storage"
	lfs "github.com/charmbracelet/charm/server/storage/local"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

// Server contains the SSH and HTTP servers required to host the Charm Cloud.
type Server struct {
	Config *config.Config
	ssh    *SSHServer
	http   *HTTPServer
}

// NewServer returns a *Server with the specified Config.
func NewServer(cfg *config.Config) (*Server, error) {
	s := &Server{Config: cfg}
	s.init(cfg)

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

func (srv *Server) init(cfg *config.Config) {
	if cfg.DB == nil {
		dp := filepath.Join(cfg.DataDir, "db")
		err := storage.EnsureDir(dp, 0o700)
		if err != nil {
			log.Fatalf("could not init sqlite path: %s", err)
		}
		db := sqlite.NewDB(filepath.Join(dp, sqlite.DbName))
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
		srv.Config = cfg.WithStats(getStatsImpl(cfg))
	}
	if cfg.JWTKeyPair == nil {
		pk, err := gossh.ParseRawPrivateKey(cfg.PrivateKey)
		if err != nil {
			log.Fatalf("could not parse private key: %s", err)
		}
		jwtKeyPair := jwt.NewJSONWebKeyPair(pk.(*ed25519.PrivateKey))
		srv.Config = cfg.WithJWTKeyPair(jwtKeyPair)
	}
}

func getStatsImpl(cfg *config.Config) stats.Stats {
	if cfg.EnableMetrics {
		return prometheus.NewStats(cfg.DB, cfg.StatsPort)
	}
	return noop.Stats{}
}
