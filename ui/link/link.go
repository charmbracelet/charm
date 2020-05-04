package link

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
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

func initialize(cc *charm.Client, code string) func() (tea.Model, tea.Cmd) {
	sp := spinner.NewModel()
	sp.ForegroundColor = "241"
	sp.Type = spinner.Dot
	return func() (tea.Model, tea.Cmd) {
		m := model{
			cc:            cc,
			lh:            newLinkHandler(),
			code:          code,
			status:        linkInit,
			alreadyLinked: false,
			err:           nil,
			spinner:       sp,
		}
		return m, handleLinkRequest(m)
	}
}

func update(msg tea.Msg, mdl tea.Model) (tea.Model, tea.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{
			err: errors.New("could not perform model assertion in update"),
		}, nil
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
		return m, tea.Quit

	case requestDeniedMsg:
		m.status = linkRequestDenied
		return m, tea.Quit

	case successMsg:
		m.status = linkSuccess
		if msg {
			m.alreadyLinked = true
		}
		return m, tea.Quit

	case timeoutMsg:
		m.status = linkTimeout
		return m, tea.Quit

	case errMsg:
		m.status = linkErr
		return m, tea.Quit

	case spinner.TickMsg:
		m.spinner, _ = spinner.Update(msg, m.spinner)
		return m, nil

	default:
		return m, nil
	}
}

func view(mdl tea.Model) string {
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

	return indent.String(fmt.Sprintf("\n%s\n", s), 2)
}

func subscriptions(mdl tea.Model) tea.Subs {
	m, ok := mdl.(model)
	if !ok {
		return nil
	}
	sub, err := spinner.MakeSub(m.spinner)
	if err != nil {
		return nil
	}
	return tea.Subs{"tick": sub}
}

// COMMANDS

func handleLinkRequest(m model) tea.Cmd {

	go func() {
		if err := m.cc.Link(m.lh, m.code); err != nil {
			m.lh.err <- err
		}
	}()

	return tea.Batch(
		handleTokenSent(m.lh),
		handleValidToken(m.lh),
		handleRequestDenied(m.lh),
		handleLinkSuccess(m.lh),
		handleTimeout(m.lh),
		handleErr(m.lh),
	)
}

func handleTokenSent(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		<-lh.tokenSent
		return tokenSentMsg{}
	}
}

func handleValidToken(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		return validTokenMsg(<-lh.validToken)
	}
}

func handleRequestDenied(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		<-lh.requestDenied
		return requestDeniedMsg{}
	}
}

func handleLinkSuccess(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		return successMsg(<-lh.success)
	}
}

func handleTimeout(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		<-lh.timeout
		return timeoutMsg{}
	}
}

func handleErr(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		return errMsg(<-lh.err)
	}
}
