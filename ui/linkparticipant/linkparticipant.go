package linkparticipant

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
	"github.com/muesli/reflow/indent"
)

// NewProgram returns a Tea program for the link participant
func NewProgram(cc *charm.Client, code string) *tea.Program {
	return tea.NewProgram(initialize(cc, code), update, view, subscriptions)
}

type status int

const (
	linkInit status = iota
	linkTokenSent
	linkTokenValid
	linkTokenInvalid
	linkRequestDenied
	linkSuccess
	linkTimeout
	linkErr
	quitting
)

// Messages
type tokenSentMsg struct{}
type validTokenMsg bool
type requestDeniedMsg struct{}
type successMsg bool
type timeoutMsg struct{}
type errMsg struct{ error }

type model struct {
	lh            *linkHandler
	cc            *charm.Client
	code          string
	status        status
	alreadyLinked bool
	err           error
}

func initialize(cc *charm.Client, code string) func() (tea.Model, tea.Cmd) {
	lh := &linkHandler{
		err: make(chan error),
	}
	return func() (tea.Model, tea.Cmd) {
		m := model{
			cc:            cc,
			lh:            lh,
			code:          code,
			status:        linkInit,
			alreadyLinked: false,
			err:           nil,
		}
		return m, nil
	}
}

func update(msg tea.Msg, mdl tea.Model) (tea.Model, tea.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return tea.ModelAssertionErr, nil
	}

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			fallthrough
		case "esc":
			fallthrough
		case "q":
			m.status = quitting
			return m, tea.Quit
		}

	case tokenSentMsg:
		m.status = linkTokenSent
		return m, nil

	case validTokenMsg:
		if msg {
			m.status = linkTokenValid
		} else {
			m.status = linkTokenInvalid
		}
		return m, nil

	case requestDeniedMsg:
		m.status = linkRequestDenied
		return m, tea.Quit

	case successMsg:
		m.status = linkSuccess
		if msg {
			m.alreadyLinked = true
		}

	case timeoutMsg:
		m.status = linkTimeout
		return m, tea.Quit

	case errMsg:
		m.status = linkErr
		return m, tea.Quit
	}

	return m, nil
}

func view(mdl tea.Model) string {
	m, ok := mdl.(model)
	if !ok {
		m.err = errors.New("could not perform assertion on model in view")
	}

	s := "Linking..."

	switch m.status {
	case linkInit:
		s += "Linking..."
	case linkTokenSent:
		s += "token sent..."
	case linkTokenValid:
		s += "token valid..."
	case linkTokenInvalid:
		s = "Invalid token."
	case linkRequestDenied:
		s = "Link request denied."
	case linkSuccess:
		s = "Linked!"
		if m.alreadyLinked {
			s += "You already linked this key, btw."
		}
	case linkTimeout:
		s = "Link request timed out. Sorry."
	case linkErr:
		s = "Error."
	case quitting:
		s = "Oh, ok. Bye."
	}

	return indent.String(fmt.Sprintf("\n%s", s), 2)
}

func subscriptions(mdl tea.Model) tea.Subs {
	return nil
}
