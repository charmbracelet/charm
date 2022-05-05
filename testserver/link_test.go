package testserver

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server"
)

var (
	cfg *server.Config
	td  string
)

func setup(t *testing.T) {
	backupFilePath := "./charm-keys-backup.tar"
	_ = os.RemoveAll(backupFilePath)
	_ = SetupTestServer(t)

	// run actual tests here
}

// create mock client
func setupTestClient(t *testing.T) *client.Client {
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
		log.Printf("unable to create new client: %s", err)
	}
	return cl
}

func ConfigFromEnv() {
	panic("unimplemented")
}

// create mock linkHandler

// TestLinkGen
func TestLinkGen(t *testing.T) {
	var linkError bool
	backupFilePath := "./charm-keys-backup.tar"
	_ = os.RemoveAll(backupFilePath)
	_ = SetupTestServer(t)
	ecfg, err := client.ConfigFromEnv()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	client1, err := client.NewClient(ecfg)
	if err != nil {
		t.Errorf("error creating first client: %v", err)
	}
	client2, err := client.NewClient(ecfg)
	if err != nil {
		t.Errorf("error creating second client: %v", err)
	}
	lc := make(chan string)
	go func() {
		lh := &linkHandler{desc: "client1", linkChan: lc}
		client1.LinkGen(lh)
	}()
	tok := <-lc
	lh := &linkHandler{desc: "client1", linkChan: lc}
	client2.Link(lh, tok)
	if linkError {
		t.Errorf("got a link error: %v", err)
	}
}

type linkHandler struct {
	desc     string
	linkChan chan string
}

func (lh *linkHandler) TokenCreated(l *proto.Link) {
	lh.printDebug("token created", l)
	lh.linkChan <- string(l.Token)
	lh.printDebug("token created sent to chan", l)
}

func (lh *linkHandler) TokenSent(l *proto.Link) {
	lh.printDebug("token sent", l)
}

func (lh *linkHandler) ValidToken(l *proto.Link) {
	lh.printDebug("valid token", l)
}

func (lh *linkHandler) InvalidToken(l *proto.Link) {
	lh.printDebug("invalid token", l)
}

func (lh *linkHandler) Request(l *proto.Link) bool {
	lh.printDebug("request", l)
	return true
}

func (lh *linkHandler) RequestDenied(l *proto.Link) {
	lh.printDebug("request denied", l)
}

func (lh *linkHandler) SameUser(l *proto.Link) {
	lh.printDebug("same user", l)
}

func (lh *linkHandler) Success(l *proto.Link) {
	lh.printDebug("success", l)
}

func (lh *linkHandler) Timeout(l *proto.Link) {
	lh.printDebug("timeout", l)
}

func (lh linkHandler) Error(l *proto.Link) {
	lh.printDebug("error", l)
}

func (lh *linkHandler) printDebug(msg string, l *proto.Link) {
	log.Printf("%s %s:\t%v\n", lh.desc, msg, l)
}
