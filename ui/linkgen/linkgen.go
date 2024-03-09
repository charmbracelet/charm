package linkgen

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui/charmclient"
	"github.com/charmbracelet/charm/ui/common"
)

type status int

const (
	initCharmClient status = iota // we're creating charm client
	linkInit                      // we're initializing the linking process
	linkTokenCreated
	linkRequested
	linkSuccess
	linkRequestDenied
	linkTimedOut
	linkError
	quitting
)

type (
	linkTokenCreatedMsg string
	linkRequestMsg      linkRequest
	linkSuccessMsg      bool // true if this account's already been linked
	linkTimeoutMsg      struct{}
)

type errMsg struct {
	err error
}

// NewProgram is a simple wrapper for tea.NewProgram for use in standalone
// mode. Pass the Charm configuration and the name of the parent command upon
// which this TUI is implemented.
func NewProgram(cfg *client.Config, parentName string) *tea.Program {
	m := NewModel(cfg)
	m.standalone = true
	m.parentName = parentName
	return tea.NewProgram(m)
}

// Model is the tea model for the link initiator program.
type Model struct {
	lh            *linkHandler
	standalone    bool           // true if this is running stadalone
	cfg           *client.Config // only applicable in standalone mode
	parentName    string         // name of the parent command used in instructional text
	styles        common.Styles
	Quit          bool // the user wants to exit the whole program
	Exit          bool // the user wants to exit this mini-app
	err           error
	status        status
	alreadyLinked bool
	token         string
	linkRequest   linkRequest
	cc            *client.Client
	buttonIndex   int // focused state of ok/cancel buttons
	spinner       spinner.Model
}

// acceptRequest rejects the current linking request.
func (m Model) acceptRequest() (Model, tea.Cmd) { // nolint: unparam
	m.lh.response <- true
	return m, nil
}

// rejectRequest rejects the current linking request.
func (m Model) rejectRequest() (Model, tea.Cmd) {
	m.lh.response <- false
	m.status = linkRequestDenied
	if m.standalone {
		return m, tea.Quit
	}

	return m, nil
}

// NewModel returns a new Model in its initial state.
func NewModel(cfg *client.Config) Model {
	lh := &linkHandler{
		err:      make(chan error),
		token:    make(chan charm.Token),
		request:  make(chan linkRequest),
		response: make(chan bool),
		success:  make(chan bool),
		timeout:  make(chan struct{}),
	}

	return Model{
		lh:            lh,
		standalone:    false,
		parentName:    "charm",
		styles:        common.DefaultStyles(),
		cfg:           cfg,
		Quit:          false,
		Exit:          false,
		err:           nil,
		status:        linkInit,
		alreadyLinked: false,
		token:         "",
		linkRequest:   linkRequest{},
		buttonIndex:   0,
		spinner:       common.NewSpinner(),
	}
}

// SetCharmClient sets the charm client on the Model.
func (m *Model) SetCharmClient(cc *client.Client) {
	if cc == nil {
		panic("charm client is nil")
	}
	m.cc = cc
}

// Init is the Bubble Tea program's initialization function. This is used in
// standalone mode.
func (m Model) Init() tea.Cmd {
	return tea.Batch(charmclient.NewClient(m.cfg), m.spinner.Tick)
}

// Update is the Tea update loop.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// General keybindings
		case "ctrl+c":
			if m.standalone {
				m.status = quitting
				return m, tea.Quit
			}
			m.Quit = true
			return m, nil
		case "q", "esc":
			if m.standalone {
				m.status = quitting
				return m, tea.Quit
			}
			m.Exit = true
			return m, nil

		// State-specific keybindings
		default:
			switch m.status {
			case linkRequested:
				switch msg.String() {
				case "j", "h", "right", "tab":
					m.buttonIndex++
					if m.buttonIndex > 1 {
						m.buttonIndex = 0
					}
				case "k", "l", "left", "shift+tab":
					m.buttonIndex--
					if m.buttonIndex < 0 {
						m.buttonIndex = 1
					}
				case "enter":
					if m.buttonIndex == 0 {
						return m.acceptRequest()
					}
					return m.rejectRequest()
				case "y":
					return m.acceptRequest()
				case "n":
					return m.rejectRequest()
				}
				return m, nil

			case linkSuccess, linkRequestDenied, linkTimedOut:
				// Any key exits
				m.Exit = true
				return m, nil
			}
		}

	case charmclient.NewClientMsg:
		m.cc = msg
		m.status = linkInit
		return m, InitLinkGen(m)

	case charmclient.ErrMsg:
		// Unknown error: fatal
		m.err = msg.Err
		return m, tea.Quit

	case charmclient.SSHAuthErrorMsg:
		m.err = msg.Err
		return m, tea.Quit

	case errMsg:
		m.status = linkError
		m.err = msg.err
		return m, nil

	case linkTokenCreatedMsg:
		m.status = linkTokenCreated
		m.token = string(msg)
		return m, nil

	case linkRequestMsg:
		m.status = linkRequested
		m.linkRequest = linkRequest(msg)
		return m, nil

	case linkSuccessMsg:
		m.status = linkSuccess
		m.alreadyLinked = bool(msg)
		if m.standalone {
			return m, tea.Quit
		}
		return m, nil

	case linkTimeoutMsg:
		m.status = linkTimedOut
		if m.standalone {
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		switch m.status {
		case initCharmClient, linkInit:
			newSpinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = newSpinnerModel
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

func (m Model) preambleView() string {
	return m.styles.Wrap.Render(fmt.Sprintf(
		"You can %s the SSH keys on another machine to your Charm account so both machines have access to your stuff. Keys can be unlinked at any time.",
		m.styles.Keyword.Render("link"),
	)) + "\n\n"
}

// View renders the UI.
func (m Model) View() string {
	var s string

	switch m.status {
	case initCharmClient:
		s += m.preambleView()
		s += m.spinner.View() + " Initializing..."
	case linkInit:
		s += m.preambleView()
		s += m.spinner.View() + " Generating link..."
	case linkTokenCreated:
		s += m.preambleView()
		s += m.styles.Wrap.Render("To link, run the following command on your other machine:") + "\n\n"
		s += m.styles.Code.Render(m.parentName+" link "+m.token) + "\n\n"
		s += common.HelpView("To cancel, press escape")
	case linkRequested:
		var d []string
		s += m.preambleView()
		s += "Link request from:\n\n"
		d = append(d, []string{"IP", m.linkRequest.requestAddr}...)
		if len(m.linkRequest.pubKey) > 50 {
			d = append(d, []string{"Key", m.linkRequest.pubKey[0:50] + "..."}...)
		}
		s += common.KeyValueView(d...)
		s += "\n\nLink this device?\n\n"
		s += common.YesButtonView(m.buttonIndex == 0) + " "
		s += common.NoButtonView(m.buttonIndex == 1)
	case linkError:
		s += m.preambleView()
		s += "Uh oh: " + m.err.Error()
	case linkSuccess:
		s += m.styles.Keyword.Render("Linked!")
		if m.alreadyLinked {
			s += " This key is already linked, btw."
		}
		if m.standalone {
			s += "\n"
		} else {
			s = m.preambleView() + s + common.HelpView("\n\nPress any key to exit...")
		}
	case linkRequestDenied:
		s += "Link request " + m.styles.Keyword.Render("denied") + "."
		if m.standalone {
			s += "\n"
		} else {
			s = m.preambleView() + s + common.HelpView("\n\nPress any key to exit...")
		}
	case linkTimedOut:
		s += m.preambleView()
		s += "Link request timed out."
		if m.standalone {
			s += "\n"
		} else {
			s += common.HelpView("Press any key to exit...")
		}
	case quitting:
		s += "Linking canceled.\n"
	}

	if m.standalone {
		s = m.styles.App.Render(s)
	}
	return s
}

// COMMANDS

// InitLinkGen runs the necessary commands for starting the link generation
// process.
func InitLinkGen(m Model) tea.Cmd {
	return tea.Batch(append(HandleLinkRequest(m), m.spinner.Tick)...)
}

// HandleLinkRequest returns a bunch of blocking commands that resolve on link
// request states. As a Tea command, this should be treated as batch:
//
//	tea.Batch(HandleLinkRequest(model)...)
func HandleLinkRequest(m Model) []tea.Cmd {
	go func() {
		if err := m.cc.LinkGen(m.lh); err != nil {
			m.lh.err <- err
		}
	}()

	// We use a series of blocking commands to interface with channels on the
	// link handler.
	return []tea.Cmd{
		generateLink(m.lh),
		handleLinkRequest(m.lh),
		handleLinkSuccess(m.lh),
		handleLinkTimeout(m.lh),
		handleLinkError(m.lh),
	}
}

// generateLink waits for either a link to be generated, or an error.
func generateLink(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		select {
		case err := <-lh.err:
			return errMsg{err}
		case tok := <-lh.token:
			return linkTokenCreatedMsg(tok)
		}
	}
}

// handleLinkRequest waits for a link request code.
func handleLinkRequest(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		return linkRequestMsg(<-lh.request)
	}
}

// handleLinkSuccess waits for data in the link success channel.
func handleLinkSuccess(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		return linkSuccessMsg(<-lh.success)
	}
}

// handleLinkTimeout waits for a timeout in the linking process.
func handleLinkTimeout(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		<-lh.timeout
		return linkTimeoutMsg{}
	}
}

// handleLinkError responds when a linking error is reported.
func handleLinkError(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		return errMsg{<-lh.err}
	}
}
