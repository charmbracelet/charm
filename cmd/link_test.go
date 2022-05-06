package cmd

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/charm/testserver"
)

var (
	cfg *server.Config
	td  string
)

func setup(t *testing.T) {
	backupFilePath := "./charm-keys-backup.tar"
	_ = os.RemoveAll(backupFilePath)
	_ = testserver.SetupTestServer(t)

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

// create mock linkHandlerTest

// TestLinkGen
func TestLinkGen(t *testing.T) {
	_ = testserver.SetupTestServer(t)
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
	t.Run("link client 1", func(t *testing.T) {
		lh := &linkHandlerTest{desc: "client1", linkChan: lc}
		err := client1.LinkGen(lh)
		if err != nil {
			t.Fatalf("failed to link in go routine: %v", err)
		}
	})
	tok := <-lc
	lh := &linkHandlerTest{desc: "client1", linkChan: lc}
	err = client2.Link(lh, tok)
	if err != nil {
		t.Fatalf("failed to link in go routine: %v", err)
	}
}

type linkHandlerTest struct {
	desc     string
	linkChan chan string
}

func (lh *linkHandlerTest) TokenCreated(l *proto.Link) {
	lh.printDebug("token created", l)
	lh.linkChan <- string(l.Token)
	lh.printDebug("token created sent to chan", l)
}

func (lh *linkHandlerTest) TokenSent(l *proto.Link) {
	lh.printDebug("token sent", l)
}

func (lh *linkHandlerTest) ValidToken(l *proto.Link) {
	lh.printDebug("valid token", l)
}

func (lh *linkHandlerTest) InvalidToken(l *proto.Link) {
	lh.printDebug("invalid token", l)
}

func (lh *linkHandlerTest) Request(l *proto.Link) bool {
	lh.printDebug("request", l)
	return true
}

func (lh *linkHandlerTest) RequestDenied(l *proto.Link) {
	lh.printDebug("request denied", l)
}

func (lh *linkHandlerTest) SameUser(l *proto.Link) {
	lh.printDebug("same user", l)
}

func (lh *linkHandlerTest) Success(l *proto.Link) {
	lh.printDebug("success", l)
}

func (lh *linkHandlerTest) Timeout(l *proto.Link) {
	lh.printDebug("timeout", l)
}

func (lh linkHandlerTest) Error(l *proto.Link) {
	lh.printDebug("error", l)
}

func (lh *linkHandlerTest) printDebug(msg string, l *proto.Link) {
	log.Printf("%s %s:\t%v\n", lh.desc, msg, l)
}
