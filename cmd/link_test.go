package cmd

import (
	"log"
	"testing"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/charm/testserver"
)

// TestValidLinkGen
func TestValidLinkGen(t *testing.T) {
	client1 := testserver.SetupTestServer(t)
	client2, err := client.NewClientWithDefaults()
	if err != nil {
		t.Fatalf("error creating second client: %v", err)
	}
	lc := make(chan string, 1)
	t.Run("link client 1", func(t *testing.T) {
		t.Parallel()
		lh := &linkHandlerTest{desc: "client1", linkChan: lc, approve: true}
		// pass testing.t to it, assert error
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

// TestInvalidLinkGen
func TestInvalidLinkGen(t *testing.T) {
	client1 := testserver.SetupTestServer(t)
	client2, err := client.NewClientWithDefaults()
	if err != nil {
		t.Fatalf("error creating second client: %v", err)
	}
	lc := make(chan string, 1)
	t.Run("link client 1", func(t *testing.T) {
		t.Parallel()
		lh := &linkHandlerTest{desc: "client1", linkChan: lc, approve: false}
		// pass testing.t to it, assert error
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
		if lh.status != requestDenied {
			t.Fatalf("expected request denied, got: %d", lh.status)
		}
	})
}

// TestTimeoutLink
func TestTimeoutLink(t *testing.T) {
	client1 := testserver.SetupTestServer(t, func(c *server.Config) *server.Config {
		return c.WithLinkTimeout(5 * time.Second)
	})
	lc := make(chan string, 1)
	t.Run("link client 1", func(t *testing.T) {
		t.Parallel()
		lh := &linkHandlerTest{desc: "client1", linkChan: lc, approve: true}
		// pass testing.t to it, assert error
		err := client1.LinkGen(lh)
		if err != nil {
			t.Fatalf("failed to link client 1: %v", err)
		}
		if lh.status != timedout {
			t.Fatalf("expected link to timeout, got: %v", lh.status)
		}
	})
}

// use these status codes for assertions in tests
type statusCode uint

const (
	ok statusCode = iota
	timedout
	invalidToken
	requestDenied
)

type linkHandlerTest struct {
	desc     string
	linkChan chan string
	approve  bool
	status   statusCode
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
	lh.status = invalidToken
}

func (lh *linkHandlerTest) Request(l *proto.Link) bool {
	return lh.approve
}

func (lh *linkHandlerTest) RequestDenied(l *proto.Link) {
	lh.status = requestDenied
}

func (lh *linkHandlerTest) SameUser(l *proto.Link) {
	lh.printDebug("same user", l)
}

func (lh *linkHandlerTest) Success(l *proto.Link) {
	lh.printDebug("success", l)
	lh.status = ok
}

func (lh *linkHandlerTest) Timeout(l *proto.Link) {
	lh.status = timedout
}

func (lh linkHandlerTest) Error(l *proto.Link) {
	lh.printDebug("error", l)
}

func (lh *linkHandlerTest) printDebug(msg string, l *proto.Link) {
	log.Printf("%s %s:\t%v\n", lh.desc, msg, l)
}
