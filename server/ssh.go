package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server/db"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
	"github.com/golang-jwt/jwt/v4"
	"github.com/muesli/toktok"
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
	config       *Config
	db           db.DB
	tokenBucket  *toktok.Bucket
	linkRequests map[charm.Token]chan *charm.Link
	server       *ssh.Server
	errorLog     *log.Logger
}

// NewSSHServer creates a new SSHServer from the provided Config.
func NewSSHServer(cfg *Config) (*SSHServer, error) {
	s := &SSHServer{
		config:   cfg,
		errorLog: cfg.errorLog,
	}
	if s.errorLog == nil {
		s.errorLog = log.Default()
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.SSHPort)
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
func (me *SSHServer) Start() {
	log.Printf("Starting SSH server on %s", me.server.Addr)
	log.Fatal(me.server.ListenAndServe())
}

// Shutdown gracefully shuts down the SSH server.
func (me *SSHServer) Shutdown(ctx context.Context) error {
	log.Printf("Stopping SSH server on %s", me.server.Addr)
	return me.server.Shutdown(ctx)
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

func (me *SSHServer) handleJWT(s ssh.Session) {
	var aud []string
	cmd := s.Command()
	if len(cmd) > 1 {
		aud = cmd[1:]
	} else {
		aud = []string{"charm"}
	}
	key, err := keyText(s)
	if err != nil {
		log.Println(err)
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("JWT for user %s\n", u.CharmID)
	j, err := me.newJWT(u.CharmID, aud...)
	if err != nil {
		log.Println(err)
		return
	}
	_, _ = s.Write([]byte(j))
	me.config.Stats.JWT()
}

func (me *SSHServer) newJWT(charmID string, audience ...string) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   charmID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		Issuer:    me.config.httpURL(),
		Audience:  audience,
	}
	token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, claims)
	token.Header["kid"] = me.config.jwtKeyPair.JWK.KeyID
	return token.SignedString(me.config.jwtKeyPair.PrivateKey)
}

// keyText is the base64 encoded public key for the glider.Session.
func keyText(s ssh.Session) (string, error) {
	if s.PublicKey() == nil {
		return "", fmt.Errorf("Session doesn't have public key")
	}
	kb := base64.StdEncoding.EncodeToString(s.PublicKey().Marshal())
	return fmt.Sprintf("%s %s", s.PublicKey().Type(), kb), nil
}
