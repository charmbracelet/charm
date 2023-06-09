package link

import (
	"errors"

	charm "github.com/charmbracelet/charm/proto"
)

type linkHandler struct {
	tokenSent     chan struct{}
	validToken    chan bool
	success       chan bool // true if the key was already linked
	requestDenied chan struct{}
	timeout       chan struct{}
	err           chan error
}

func newLinkHandler() *linkHandler {
	return &linkHandler{
		tokenSent:     make(chan struct{}),
		validToken:    make(chan bool),
		success:       make(chan bool),
		requestDenied: make(chan struct{}),
		timeout:       make(chan struct{}),
		err:           make(chan error),
	}
}

func (lh *linkHandler) TokenCreated(_ *charm.Link) {
	// Not implemented for the link participant
}

func (lh *linkHandler) TokenSent(_ *charm.Link) {
	lh.tokenSent <- struct{}{}
}

func (lh *linkHandler) ValidToken(_ *charm.Link) {
	lh.validToken <- true
}

func (lh *linkHandler) InvalidToken(_ *charm.Link) {
	lh.validToken <- false
}

func (lh *linkHandler) Request(_ *charm.Link) bool {
	// Not implemented for the link participant
	return false
}

func (lh *linkHandler) RequestDenied(_ *charm.Link) {
	lh.requestDenied <- struct{}{}
}

func (lh *linkHandler) SameUser(_ *charm.Link) {
	lh.success <- true
}

func (lh *linkHandler) Success(_ *charm.Link) {
	lh.success <- false
}

func (lh *linkHandler) Timeout(_ *charm.Link) {
	lh.timeout <- struct{}{}
}

func (lh *linkHandler) Error(_ *charm.Link) {
	lh.err <- errors.New("error")
}
