package linkparticipant

import "github.com/charmbracelet/charm"

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

func (lh *linkHandler) TokenCreated(l *charm.Link)  {}
func (lh *linkHandler) TokenSent(l *charm.Link)     {}
func (lh *linkHandler) ValidToken(l *charm.Link)    {}
func (lh *linkHandler) InvalidToken(l *charm.Link)  {}
func (lh *linkHandler) Request(l *charm.Link)       {}
func (lh *linkHandler) RequestDenied(l *charm.Link) {}
func (lh *linkHandler) SameAccount(l *charm.Link)   {}
func (lh *linkHandler) Success(l *charm.Link)       {}
func (lh *linkHandler) Timeout(l *charm.Link)       {}
func (lh *linkHandler) Error(l *charm.Link)         {}
