package server

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server/db"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
	"github.com/muesli/toktok"
	gossh "golang.org/x/crypto/ssh"
)

// Session represents a Charm User's SSH session.
type Session struct {
	ssh.Session
}

// SessionHandler defines a function that handles a session for a given SSH
// command.
type SessionHandler func(s Session)

// SSHServer serves the SSH protocol and handles requests to authenticate and
// link Charm user accounts.
type SSHServer struct {
	config        *Config
	db            db.DB
	tokenBucket   *toktok.Bucket
	linkRequests  map[charm.Token]chan *charm.Link
	jwtPrivateKey *rsa.PrivateKey
	server        *ssh.Server
}

// NewSSHServer creates a new SSHServer from the provided Config.
func NewSSHServer(cfg *Config) (*SSHServer, error) {
	s := &SSHServer{config: cfg}
	addr := fmt.Sprintf(":%d", cfg.SSHPort)
	pk, err := gossh.ParseRawPrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	s.jwtPrivateKey = pk.(*rsa.PrivateKey)
	b, err := toktok.NewBucket(6)
	if err != nil {
		return nil, err
	}
	s.tokenBucket = &b
	s.db = cfg.DB
	s.linkRequests = make(map[charm.Token]chan *charm.Link)
	srv, err := wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPEM(cfg.PrivateKey),
		wish.WithPublicKeyAuth(s.authHandler),
		wish.WithMiddleware(
			s.sshMiddleware(),
		),
	)
	if err != nil {
		return nil, err
	}
	s.server = srv
	return s, nil
}

// Start serves the SSH protocol on the configured port.
func (me *SSHServer) Start(ctx context.Context) {
	log.Printf("Starting SSH server on %s", me.server.Addr)
	go func() {
		err := me.server.ListenAndServe()
		if err != nil && err != context.Canceled && err != ssh.ErrServerClosed {
			log.Fatalf("ssh server crashed: %s", err)
		}
	}()

	<-ctx.Done()
	if err := me.server.Shutdown(ctx); err != context.Canceled {
		log.Printf("unexpected error shutting down ssh server: %s", err)
	}

	log.Println("SSH server stopped")
}

func (me *SSHServer) sendAPIMessage(s ssh.Session, msg string) error {
	return me.sendJSON(s, charm.Message{Message: msg})
}

func (me *SSHServer) sendJSON(s ssh.Session, o interface{}) error {
	return json.NewEncoder(s).Encode(o)
}

func (me *SSHServer) authHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	return true
}

// keyText is the base64 encoded public key for the glider.Session.
func keyText(s ssh.Session) (string, error) {
	if s.PublicKey() == nil {
		return "", fmt.Errorf("Session doesn't have public key")
	}
	kb := base64.StdEncoding.EncodeToString(s.PublicKey().Marshal())
	return fmt.Sprintf("%s %s", s.PublicKey().Type(), kb), nil
}
