package server

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/keygen"
)

func TestSSHAuthMiddleware(t *testing.T) {
	cfg := DefaultConfig()
	td := t.TempDir()
	cfg.DataDir = filepath.Join(td, ".data")
	sp := filepath.Join(td, ".ssh")
	kp, err := keygen.NewWithWrite(filepath.Join(sp, "charm_server"), []byte(""), keygen.Ed25519)
	if err != nil {
		t.Fatalf("keygen error: %s", err)
	}
	cfg = cfg.WithKeys(kp.PublicKey(), kp.PrivateKeyPEM())
	s, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("new server error: %s", err)
	}

	go s.Start()
	t.Run("health-ping", func(t *testing.T) {
		_, err := fetchURL(fmt.Sprintf("http://localhost:%d", cfg.HealthPort), 3)
		if err != nil {
			t.Fatal(fmt.Sprintf("could not ping server: %s", err))
		}
	})
	t.Run("api-auth", func(t *testing.T) {
		ccfg, err := client.ConfigFromEnv()
		if err != nil {
			t.Fatalf("client config from env error: %s", err)
		}
		ccfg.Host = cfg.Host
		ccfg.SSHPort = cfg.SSHPort
		ccfg.HTTPPort = cfg.HTTPPort
		ccfg.DataDir = filepath.Join(td, ".client-data")
		cl, err := client.NewClient(ccfg)
		if err != nil {
			t.Fatalf("new client error: %s", err)
		}
		auth, err := cl.Auth()
		if err != nil {
			t.Fatalf("auth error: %s", err)
		}
		if auth.JWT == "" {
			t.Fatal("auth error, missing JWT")
		}
		if auth.ID == "" {
			t.Fatal("auth error, missing ID")
		}
		if auth.PublicKey == "" {
			t.Fatal("auth error, missing PublicKey")
		}
		// if len(auth.EncryptKeys) == 0 {
		// 	t.Fatal("auth error, missing EncryptKeys")
		// }
	})
	t.Cleanup(func() {
		err := s.Close()
		if err != nil {
			log.Printf("error closing server: %s", err)
		}
	})
}

func fetchURL(url string, retries int) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		if retries > 0 {
			time.Sleep(time.Second)
			return fetchURL(url, retries-1)
		}
		return nil, err
	}
	if resp.StatusCode != 200 {
		return resp, fmt.Errorf("bad http status code: %d", resp.StatusCode)
	}
	return resp, nil
}
