package linkgen

import (
	"errors"

	charm "github.com/charmbracelet/charm/proto"
)

// linkRequest carries metadata pertaining to a link request.
type linkRequest struct {
	pubKey      string
	requestAddr string
}

// linkHandler implements the charm.LinkHandler interface.
type linkHandler struct {
	err      chan error
	token    chan charm.Token
	request  chan linkRequest
	response chan bool
	success  chan bool
	timeout  chan struct{}
}

func (lh *linkHandler) TokenCreated(l *charm.Link) {
	lh.token <- l.Token
}

func (lh *linkHandler) TokenSent(l *charm.Link) {}

func (lh *linkHandler) ValidToken(l *charm.Link) {}

func (lh *linkHandler) InvalidToken(l *charm.Link) {}

// Request handles link approvals. The remote machine sends an approval request,
// which we send to the Tea UI as a message. The Tea application then sends a
// response to the link handler's response channel with a command.
func (lh *linkHandler) Request(l *charm.Link) bool {
	lh.request <- linkRequest{l.RequestPubKey, l.RequestAddr}
	return <-lh.response
}

func (lh *linkHandler) RequestDenied(l *charm.Link) {}

// Successful link, but this account has already been linked.
func (lh *linkHandler) SameUser(l *charm.Link) {
	lh.success <- true
}

func (lh *linkHandler) Success(l *charm.Link) {
	lh.success <- false
}

func (lh *linkHandler) Timeout(l *charm.Link) {
	lh.timeout <- struct{}{}
}

func (lh *linkHandler) Error(l *charm.Link) {
	lh.err <- errors.New("there’s been an error; please try again")
}
