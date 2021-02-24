package server

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	glider "github.com/gliderlabs/ssh"
	"github.com/muesli/toktok"
	"golang.org/x/crypto/ssh"
)

const serverError = "There was an error. Please try again later."

type Session struct {
	glider.Session
}

type SessionHandler func(s Session)

type Router struct {
	routes map[string]SessionHandler
}

type SSHServer struct {
	config        Config
	db            DB
	tokenBucket   *toktok.Bucket
	linkRequests  map[Token]chan *Link
	jwtPrivateKey *rsa.PrivateKey
	router        *Router
	Server        *glider.Server
	Port          int
}

type Info struct {
	Session Session
	Host    string
	Port    int
}

func NewSSHServer(cfg Config) *SSHServer {
	s := &SSHServer{config: cfg}
	s.router = &Router{
		routes: make(map[string]SessionHandler),
	}
	s.Server = &glider.Server{
		Version:          "OpenSSH_7.6p1",
		Addr:             fmt.Sprintf(":%d", cfg.SSHPort),
		Handler:          s.sessionHandler,
		PublicKeyHandler: s.authHandler,
	}
	s.Server.SetOption(glider.HostKeyPEM(cfg.PrivateKey))
	pk, err := x509.ParsePKCS1PrivateKey(cfg.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	s.jwtPrivateKey = pk
	b, err := toktok.NewBucket(6)
	if err != nil {
		log.Fatal(err)
	}
	s.tokenBucket = &b
	s.db = cfg.DB
	s.linkRequests = make(map[Token]chan *Link)
	s.AddHandler("api-auth", s.HandleAPIAuth)
	s.AddHandler("api-keys", s.HandleAPIKeys)
	s.AddHandler("api-link", s.HandleAPILink)
	s.AddHandler("api-unlink", s.HandleAPIUnlink)
	s.AddHandler("id", s.HandleID)
	s.AddHandler("jwt", s.HandleJWT)
	return s
}

func (me *SSHServer) Start() {
	if len(me.router.routes) == 0 {
		log.Fatalf("no routes specified")
	}
	log.Printf("Starting SSH server on %s", me.Server.Addr)
	log.Fatal(me.Server.ListenAndServe())
}

func (me *SSHServer) SendAPIMessage(s Session, msg string) error {
	return me.SendJSON(s, LinkerMessage{msg})
}

func (me *SSHServer) SendJSON(s Session, o interface{}) error {
	return json.NewEncoder(s).Encode(o)
}

func (me *SSHServer) AddHandler(route string, h SessionHandler) {
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

func (r *Router) Route(route string, s Session) {
	h, ok := r.routes[route]
	if !ok {
		return
	}
	h(s)
}

func (s *Session) KeyText() (string, error) {
	if s.Session.PublicKey() == nil {
		return "", fmt.Errorf("Session doesn't have public key")
	}
	kb := base64.StdEncoding.EncodeToString(s.Session.PublicKey().Marshal())
	return fmt.Sprintf("%s %s", s.Session.PublicKey().Type(), kb), nil
}
