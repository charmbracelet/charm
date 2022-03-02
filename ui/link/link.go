package link

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/ui/charmclient"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/lipgloss"
)

var viewStyle = lipgloss.NewStyle().Padding(1, 2, 2, 3)

// NewProgram returns a Tea program for the link participant.
func NewProgram(cfg *client.Config, code string) *tea.Program {
	return tea.NewProgram(newModel(cfg, code))
}

type status int

const (
	initCharmClient status = iota
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

type (
	tokenSentMsg     struct{}
	validTokenMsg    bool
	requestDeniedMsg struct{}
	successMsg       bool
	timeoutMsg       struct{}
	errMsg           struct{ err error }
)

type model struct {
	lh            *linkHandler
	cfg           *client.Config
	cc            *client.Client
	styles        common.Styles
	code          string
	status        status
	alreadyLinked bool
	err           error
	spinner       spinner.Model
}

func newModel(cfg *client.Config, code string) model {
	return model{
		lh:            newLinkHandler(),
		cfg:           cfg,
		styles:        common.DefaultStyles(),
		code:          code,
		status:        initCharmClient,
		alreadyLinked: false,
		err:           nil,
		spinner:       common.NewSpinner(),
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
		return m, nil
	}
}

func (m model) View() string {
	if m.err != nil {
		return viewStyle.Render(m.err.Error())
	}

	s := m.spinner.View() + " "

	switch m.status {
	case initCharmClient:
		s += "Initializing..."
	case linkInit:
		s += "Linking..."
	case linkTokenSent:
		s += fmt.Sprintf("Token %s. Waiting for validation...", m.styles.Keyword.Render("sent"))
	case linkTokenValid:
		s += fmt.Sprintf("Token %s. Waiting for authorization...", m.styles.Keyword.Render("valid"))
	case linkTokenInvalid:
		s = fmt.Sprintf("%s token. Goodbye.", m.styles.Keyword.Render("Invalid"))
	case linkRequestDenied:
		s = fmt.Sprintf("Link request %s. Sorry, kid.", m.styles.Keyword.Render("denied"))
	case linkSuccess:
		s = m.styles.Keyword.Render("Linked!")
		if m.alreadyLinked {
			s += " You already linked this key, btw."
		}
	case linkTimeout:
		s = fmt.Sprintf("Link request %s. Sorry.", m.styles.Keyword.Render("timed out"))
	case linkErr:
		s = m.styles.Keyword.Render("Error.")
	case quitting:
		s = "Oh, ok. Bye."
	}

	return viewStyle.Render(s)
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
