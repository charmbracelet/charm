package link

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/charmclient"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/muesli/reflow/indent"
)

// NewProgram returns a Tea program for the link participant.
func NewProgram(cfg *charm.Config, code string) *tea.Program {
	return tea.NewProgram(newModel(cfg, code))
}

type status int

const (
	initCharmClient status = iota
	keygenRunning
	keygenFinished
	linkInit
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
type errMsg struct {
	err error
}

type model struct {
	lh            *linkHandler
	cfg           *charm.Config
	cc            *charm.Client
	code          string
	status        status
	alreadyLinked bool
	err           error
	spinner       spinner.Model
	keygen        keygen.Model
}

func newModel(cfg *charm.Config, code string) model {
	sp := spinner.NewModel()
	sp.ForegroundColor = "241"
	sp.Frames = common.SpinnerFrames

	return model{
		cfg:           cfg,
		lh:            newLinkHandler(),
		code:          code,
		status:        initCharmClient,
		alreadyLinked: false,
		err:           nil,
		spinner:       sp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		charmclient.NewClient(m.cfg),
		spinner.Tick,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.status = quitting
			return m, tea.Quit
		default:
			return m, nil
		}

	case charmclient.NewClientMsg:
		m.cc = msg
		m.status = linkInit
		return m, handleLinkRequest(m)

	case charmclient.ErrMsg:
		m.err = msg.Err
		return m, tea.Quit

	case charmclient.SSHAuthErrorMsg:
		if m.status == initCharmClient {
			m.status = keygenRunning
			m.keygen = keygen.NewModel()
			return m, keygen.GenerateKeys
		}
		m.err = msg.Err
		return m, tea.Quit

	case keygen.DoneMsg:
		m.status = keygenFinished
		return m, charmclient.NewClient(m.cfg)

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
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		if m.status == keygenRunning {
			newModel, cmd := m.keygen.Update(msg)
			keygenModel, ok := newModel.(keygen.Model)
			if !ok {
				panic("could not perform assertion on keygen model")
			}

			m.keygen = keygenModel
			return m, cmd
		}

		return m, nil
	}
}

func (m model) View() string {
	if m.err != nil {
		return paddedView(m.err.Error())
	}

	s := m.spinner.View() + " "

	switch m.status {
	case initCharmClient:
		s += "Initializing..."
	case keygenRunning:
		if m.keygen.Status != keygen.StatusSuccess {
			s += m.keygen.View()
		} else {
			s = m.keygen.View()
		}
	case linkInit:
		s += "Linking..."
	case linkTokenSent:
		s += fmt.Sprintf("Token %s. Waiting for validation...", common.Keyword("sent"))
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

	return paddedView(s)
}

func paddedView(s string) string {
	return indent.String(fmt.Sprintf("\n%s\n\n", s), 2)
}

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
		return errMsg{<-lh.err}
	}
}
