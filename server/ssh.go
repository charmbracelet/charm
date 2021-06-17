package server

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server/db"
	glider "github.com/gliderlabs/ssh"
	"github.com/muesli/toktok"
	"golang.org/x/crypto/ssh"
)

// Session represents a Charm User's SSH session.
type Session struct {
	glider.Session
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
	router        *router
	server        *glider.Server
	port          int
}

type router struct {
	routes map[string]SessionHandler
}

// NewSSHServer creates a new SSHServer from the provided Config.
func NewSSHServer(cfg *Config) (*SSHServer, error) {
	s := &SSHServer{config: cfg}
	s.router = &router{
		routes: make(map[string]SessionHandler),
	}
	s.server = &glider.Server{
		Version:          "OpenSSH_7.6p1",
		Addr:             fmt.Sprintf(":%d", cfg.SSHPort),
		Handler:          s.sessionHandler,
		PublicKeyHandler: s.authHandler,
	}
	s.server.SetOption(glider.HostKeyPEM(cfg.PrivateKey))
	pk, err := ssh.ParseRawPrivateKey(cfg.PrivateKey)
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
	s.addHandler("api-auth", s.handleAPIAuth)
	s.addHandler("api-keys", s.handleAPIKeys)
	s.addHandler("api-link", s.handleAPILink)
	s.addHandler("api-unlink", s.handleAPIUnlink)
	s.addHandler("id", s.handleID)
	s.addHandler("jwt", s.handleJWT)
	return s, nil
}

// Start serves the SSH protocol on the configured port.
func (me *SSHServer) Start() {
	if len(me.router.routes) == 0 {
		log.Fatalf("no routes specified")
	}
	log.Printf("Starting SSH server on %s", me.server.Addr)
	log.Fatal(me.server.ListenAndServe())
}

func (me *SSHServer) sendAPIMessage(s Session, msg string) error {
	return me.sendJSON(s, charm.Message{Message: msg})
}

func (me *SSHServer) sendJSON(s Session, o interface{}) error {
	return json.NewEncoder(s).Encode(o)
}

func (me *SSHServer) addHandler(route string, h SessionHandler) {
	me.router.routes[route] = h
}

func (me *SSHServer) sessionHandler(s glider.Session) {
	// s.Write([]byte("\x1b[2J\x1b[1;1H")) // TODO middleware
	var route string
	cmds := s.Command()
	if len(cmds) > 0 {
		route = cmds[0]
	}
	log.Printf("ssh %s\n", route)
	me.router.Route(route, Session{s})
}

func (me *SSHServer) authHandler(ctx glider.Context, key glider.PublicKey) bool {
	return true
}

func (me *SSHServer) passHandler(ctx glider.Context, pass string) bool {
	return false
}

func (me *SSHServer) bannerCallback(cm ssh.ConnMetadata) string {
	return fmt.Sprintf("\nHello %s put whatever you want as a password. It's no big whoop!\n\n", cm.User())
}

func (me *SSHServer) serverConfigCallback(ctx glider.Context) *ssh.ServerConfig {
	return &ssh.ServerConfig{
		BannerCallback: me.bannerCallback,
	}
}

// KeyText is the base64 encoded public key for the *Session.
func (s *Session) KeyText() (string, error) {
	if s.Session.PublicKey() == nil {
		return "", fmt.Errorf("Session doesn't have public key")
	}
	kb := base64.StdEncoding.EncodeToString(s.Session.PublicKey().Marshal())
	return fmt.Sprintf("%s %s", s.Session.PublicKey().Type(), kb), nil
}

func (r *router) Route(route string, s Session) {
	h, ok := r.routes[route]
	if !ok {
		return
	}
	h(s)
}
