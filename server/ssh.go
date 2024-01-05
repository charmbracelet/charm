package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	glog "log"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server/db"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	rm "github.com/charmbracelet/wish/recover"
	jwt "github.com/golang-jwt/jwt/v4"
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
	config    *Config
	db        db.DB
	server    *ssh.Server
	errorLog  *glog.Logger
	linkQueue charm.LinkQueue
}

// NewSSHServer creates a new SSHServer from the provided Config.
func NewSSHServer(cfg *Config) (*SSHServer, error) {
	s := &SSHServer{
		config:    cfg,
		errorLog:  cfg.errorLog,
		linkQueue: cfg.linkQueue,
	}

	if s.errorLog == nil {
		s.errorLog = log.StandardLog(log.StandardLogOptions{
			ForceLevel: log.ErrorLevel,
		})
	}
	addr := fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.SSHPort)
	s.db = cfg.DB
	if s.linkQueue == nil {
		s.linkQueue = &channelLinkQueue{
			s:            s,
			linkRequests: make(map[charm.Token]chan *charm.Link),
		}
	}
	opts := []ssh.Option{
		wish.WithAddress(addr),
		wish.WithHostKeyPEM(cfg.PrivateKey),
		wish.WithPublicKeyAuth(s.authHandler),
		wish.WithMiddleware(
			rm.MiddlewareWithLogger(
				log.NewWithOptions(os.Stderr, log.Options{Level: log.ErrorLevel}),
				s.sshMiddleware(),
			),
		),
	}
	fp := filepath.Join(cfg.DataDir, ".ssh", "authorized_keys")
	if _, err := os.Stat(fp); err == nil {
		log.Debug("Loading authorized_keys from", "path", fp)
		opts = append(opts, wish.WithAuthorizedKeys(fp))
	}
	srv, err := wish.NewServer(opts...)
	if err != nil {
		return nil, err
	}
	s.server = srv
	return s, nil
}

// Start serves the SSH protocol on the configured port.
func (me *SSHServer) Start() error {
	log.Info("Starting SSH server", "addr", me.server.Addr)
	if err := me.server.ListenAndServe(); err != ssh.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the SSH server.
func (me *SSHServer) Shutdown(ctx context.Context) error {
	log.Info("Stopping SSH server", "addr", me.server.Addr)
	return me.server.Shutdown(ctx)
}

func (me *SSHServer) sendAPIMessage(s ssh.Session, msg string) error {
	return me.sendJSON(s, charm.Message{Message: msg})
}

func (me *SSHServer) sendJSON(s ssh.Session, o interface{}) error {
	return json.NewEncoder(s).Encode(o)
}

func (me *SSHServer) authHandler(_ ssh.Context, _ ssh.PublicKey) bool {
	return true
}

func (me *SSHServer) handleJWT(s ssh.Session) {
	var aud []string
	if cmd := s.Command(); len(cmd) > 1 {
		aud = cmd[1:]
	} else {
		aud = []string{"charm"}
	}
	key, err := keyText(s)
	if err != nil {
		log.Error(err)
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("JWT for user", "id", u.CharmID)
	j, err := me.newJWT(u.CharmID, aud...)
	if err != nil {
		log.Error(err)
		return
	}
	_, _ = s.Write([]byte(j))
	me.config.Stats.JWT()
}

func (me *SSHServer) newJWT(charmID string, audience ...string) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   charmID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		Issuer:    me.config.httpURL().String(),
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
