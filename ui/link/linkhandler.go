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

func (lh *linkHandler) TokenCreated(l *charm.Link) {
	// Not implemented for the link participant
}

func (lh *linkHandler) TokenSent(l *charm.Link) {
	lh.tokenSent <- struct{}{}
}

func (lh *linkHandler) ValidToken(l *charm.Link) {
	lh.validToken <- true
}

func (lh *linkHandler) InvalidToken(l *charm.Link) {
	lh.validToken <- false
}

func (lh *linkHandler) Request(l *charm.Link) bool {
	// Not implemented for the link participant
	return false
}

func (lh *linkHandler) RequestDenied(l *charm.Link) {
	lh.requestDenied <- struct{}{}
}

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
	lh.err <- errors.New("error")
}
