package cmd

import (
	"log"
	"testing"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/testserver"
)

// TestLinkGen
func TestLinkGen(t *testing.T) {
	_ = testserver.SetupTestServer(t)
	ecfg, err := client.ConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	client1, err := client.NewClient(ecfg)
	if err != nil {
		t.Fatalf("error creating first client: %v", err)
	}
	client2, err := client.NewClient(ecfg)
	if err != nil {
		t.Fatalf("error creating second client: %v", err)
	}
	lc := make(chan string, 1)
	t.Run("link client 1", func(t *testing.T) {
		t.Parallel()
		lh := &linkHandlerTest{desc: "client1", linkChan: lc}
		err := client1.LinkGen(lh)
		if err != nil {
			t.Fatalf("failed to link client 1: %v", err)
		}
	})
	t.Run("link client 2", func(t *testing.T) {
		t.Parallel()
		tok := <-lc
		lh := &linkHandlerTest{desc: "client2", linkChan: lc}
		err = client2.Link(lh, tok)
		if err != nil {
			t.Fatalf("failed to link client 2: %v", err)
		}
	})
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
