package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/meowgorithm/babylogger"
	"goji.io"
	"goji.io/pat"
)

// TavernServer is the HTTP server for Charm public file server.
type TavernServer struct {
	handler     http.Handler
	uploadsPath string
	config      *Config
}

func NewTavernServer(cfg *Config) *TavernServer {
	uploadsPath := filepath.Join(".", cfg.DataDir, "tavern")

	mux := goji.NewMux()
	mux.Use(babylogger.Middleware)

	fs := http.FileServer(http.Dir(uploadsPath))
	mux.Handle(pat.Get("/*"), fs)

	ts := &TavernServer{config: cfg, uploadsPath: uploadsPath, handler: mux}

	return ts
}

func (s *TavernServer) Start() error {
	err := os.MkdirAll(s.uploadsPath, os.ModePerm)
	if err != nil {
		return err
	}

	listenAddr := fmt.Sprintf(":%s", s.config.TavernPort)
	srv := &http.Server{
		Addr:    listenAddr,
		Handler: s.handler,
	}

	log.Printf("Tavern server listening on %s, serving %s", listenAddr, s.uploadsPath)
	return srv.ListenAndServe()
}
