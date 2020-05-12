package link

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/indent"
)

// NewProgram returns a Boba program for the link participant
func NewProgram(cc *charm.Client, code string) *boba.Program {
	return boba.NewProgram(initialize(cc, code), update, view)
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

type tokenSentMsg struct{}
type validTokenMsg bool
type requestDeniedMsg struct{}
type successMsg bool
type timeoutMsg struct{}
type errMsg error

type model struct {
	lh            *linkHandler
	cc            *charm.Client
	code          string
	status        status
	alreadyLinked bool
	err           error
	spinner       spinner.Model
}

func initialize(cc *charm.Client, code string) func() (boba.Model, boba.Cmd) {
	sp := spinner.NewModel()
	sp.ForegroundColor = "241"
	sp.Type = spinner.Dot
	return func() (boba.Model, boba.Cmd) {
		m := model{
			cc:            cc,
			lh:            newLinkHandler(),
			code:          code,
			status:        linkInit,
			alreadyLinked: false,
			err:           nil,
			spinner:       sp,
		}
		return m, boba.Batch(
			handleLinkRequest(m),
			spinner.Tick(sp),
		)
	}
}

func update(msg boba.Msg, mdl boba.Model) (boba.Model, boba.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{
			err: errors.New("could not perform model assertion in update"),
		}, nil
	}

	switch msg := msg.(type) {

	case boba.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			fallthrough
		case "esc":
			fallthrough
		case "q":
			m.status = quitting
			return m, boba.Quit
		default:
			return m, nil
		}

	case tokenSentMsg:
		m.status = linkTokenSent
		return m, nil

	case validTokenMsg:
		if msg {
			m.status = linkTokenValid
			return m, nil
		}
		m.status = linkTokenInvalid
		return m, boba.Quit

	case requestDeniedMsg:
		m.status = linkRequestDenied
		return m, boba.Quit

	case successMsg:
		m.status = linkSuccess
		if msg {
			m.alreadyLinked = true
		}
		return m, boba.Quit

	case timeoutMsg:
		m.status = linkTimeout
		return m, boba.Quit

	case errMsg:
		m.status = linkErr
		return m, boba.Quit

	case spinner.TickMsg:
		var cmd boba.Cmd
		m.spinner, cmd = spinner.Update(msg, m.spinner)
		return m, cmd

	default:
		return m, nil
	}
}

func view(mdl boba.Model) string {
	m, ok := mdl.(model)
	if !ok {
		m.err = errors.New("could not perform assertion on model in view")
	}

	s := spinner.View(m.spinner) + " "

	switch m.status {
	case linkInit:
		s += "Linking..."
	case linkTokenSent:
		s += "Token sent..."
	case linkTokenValid:
		s += fmt.Sprintf("Token %s. Waiting for authorization...", common.Keyword("valid"))
	case linkTokenInvalid:
		s = fmt.Sprintf("%s token. Goodbye.", common.Keyword("Invalid"))
	case linkRequestDenied:
		s = fmt.Sprintf("Link request %s. Sorry, kid.", common.Keyword("denied"))
	case linkSuccess:
		s = common.Keyword("Linked!")
		if m.alreadyLinked {
			s += " You already linked this key, btw."
		}
	case linkTimeout:
		s = fmt.Sprintf("Link request %s. Sorry.", common.Keyword("timed out"))
	case linkErr:
		s = common.Keyword("Error.")
	case quitting:
		s = "Oh, ok. Bye."
	}

	return indent.String(fmt.Sprintf("\n%s\n\n", s), 2)
}

// COMMANDS

func handleLinkRequest(m model) boba.Cmd {

	go func() {
		if err := m.cc.Link(m.lh, m.code); err != nil {
			m.lh.err <- err
		}
	}()

	return boba.Batch(
		handleTokenSent(m.lh),
		handleValidToken(m.lh),
		handleRequestDenied(m.lh),
		handleLinkSuccess(m.lh),
		handleTimeout(m.lh),
		handleErr(m.lh),
	)
}

func handleTokenSent(lh *linkHandler) boba.Cmd {
	return func() boba.Msg {
		<-lh.tokenSent
		return tokenSentMsg{}
	}
}

func handleValidToken(lh *linkHandler) boba.Cmd {
	return func() boba.Msg {
		return validTokenMsg(<-lh.validToken)
	}
}

func handleRequestDenied(lh *linkHandler) boba.Cmd {
	return func() boba.Msg {
		<-lh.requestDenied
		return requestDeniedMsg{}
	}
}

func handleLinkSuccess(lh *linkHandler) boba.Cmd {
	return func() boba.Msg {
		return successMsg(<-lh.success)
	}
}

func handleTimeout(lh *linkHandler) boba.Cmd {
	return func() boba.Msg {
		<-lh.timeout
		return timeoutMsg{}
	}
}

func handleErr(lh *linkHandler) boba.Cmd {
	return func() boba.Msg {
		return errMsg(<-lh.err)
	}
}
