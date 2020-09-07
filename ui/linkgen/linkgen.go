package linkgen

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/charmclient"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/muesli/reflow/indent"
)

type status int

const (
	initCharmClient status = iota // we're creating charm client
	keygenRunning
	keygenFinished
	linkInit // we're initializing the linking process
	linkTokenCreated
	linkRequested
	linkSuccess
	linkRequestDenied
	linkTimedOut
	linkError
	quitting
)

type linkTokenCreatedMsg string
type linkRequestMsg linkRequest
type linkSuccessMsg bool // true if this account's already been linked
type linkTimeoutMsg struct{}
type errMsg struct {
	err error
}

// NewProgram is a simple wrapper for tea.NewProgram. For use in standalone
// mode.
func NewProgram(cfg *charm.Config) *tea.Program {
	return tea.NewProgram(Init(cfg), Update, View)
}

// Model is the tea model for the link initiator program.
type Model struct {
	lh            *linkHandler
	standalone    bool          // true if this is running as a stadalone tea program
	cfg           *charm.Config // only applicable in standalone mode
	Quit          bool          // indicates the user wants to exit the whole program
	Exit          bool          // indicates the user wants to exit this mini-app
	err           error
	status        status
	alreadyLinked bool
	token         string
	linkRequest   linkRequest
	cc            *charm.Client
	buttonIndex   int // focused state of ok/cancel buttons
	spinner       spinner.Model
	keygen        keygen.Model
}

// acceptRequest rejects the current linking request.
func (m Model) acceptRequest() (Model, tea.Cmd) {
	m.lh.response <- true
	return m, nil
}

// rejectRequset rejects the current linking request.
func (m Model) rejectRequest() (Model, tea.Cmd) {
	m.lh.response <- false
	m.status = linkRequestDenied
	if m.standalone {
		return m, tea.Quit
	}

	return m, nil
}

// NewModel returns a new Model in its initial state.
func NewModel() Model {
	lh := &linkHandler{
		err:      make(chan error),
		token:    make(chan string),
		request:  make(chan linkRequest),
		response: make(chan bool),
		success:  make(chan bool),
		timeout:  make(chan struct{}),
	}

	s := spinner.NewModel()
	s.Frames = spinner.Dot
	s.ForegroundColor = "241"

	return Model{
		lh:            lh,
		standalone:    false,
		Quit:          false,
		Exit:          false,
		err:           nil,
		status:        linkInit,
		alreadyLinked: false,
		token:         "",
		linkRequest:   linkRequest{},
		buttonIndex:   0,
		spinner:       s,
	}
}

// SetCharmClient sets the charm client on the Model.
func (m *Model) SetCharmClient(cc *charm.Client) {
	if cc == nil {
		panic("charm client is nil")
	}
	m.cc = cc
}

// Init is the Bubble Tea program's initialization function. This is used in
// standalone mode.
func Init(cfg *charm.Config) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		m := NewModel()
		m.status = initCharmClient
		m.standalone = true
		m.cfg = cfg

		return m, tea.Batch(charmclient.NewClient(cfg), spinner.Tick(m.spinner))
	}
}

// Update is the Tea update loop.
func Update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m, ok := model.(Model)
	if !ok {
		m.err = errors.New("could not perform model assertion in update")
	}

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
		if m.status == initCharmClient {
			// SSH auth didn't work, so let's try generating keys
			m.status = keygenRunning
			m.keygen = keygen.NewModel()
			return m, keygen.GenerateKeys
		}
		// We tried the keygen and it still didn't work: fatal
		m.err = msg.Err
		return m, tea.Quit

	case keygen.DoneMsg:
		// The keygen's finished, so let's try creating a Charm Client again
		m.status = keygenFinished
		return m, charmclient.NewClient(m.cfg)

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
		case initCharmClient, keygenRunning, linkInit:
			newSpinnerModel, cmd := spinner.Update(msg, m.spinner)
			m.spinner = newSpinnerModel
			return m, cmd
		}
		return m, nil
	}

	if m.status == keygenRunning {
		newKeygenModel, cmd := keygen.Update(msg, m.keygen)
		mdl, ok := newKeygenModel.(keygen.Model)
		if !ok {
			// This shouldn't happen, but if it does, it's fatal
			m.err = errors.New("could not assert model to keygen.Model in linkgen update")
			return m, tea.Quit
		}
		m.keygen = mdl
		return m, cmd
	}

	return m, nil
}

// View renders the UI.
func View(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		m.status = linkError
		m.err = errors.New("could not perform model assertion in view")
	}

	var s string
	preamble := common.Wrap(fmt.Sprintf(
		"You can %s the SSH keys on another machine to your Charm account so both machines have access to your stuff. You can unlink keys at any time.\n\n",
		common.Keyword("link"),
	))

	switch m.status {
	case initCharmClient:
		s += preamble
		s += spinner.View(m.spinner) + " Initializing..."
	case keygenRunning:
		s += preamble
		if m.keygen.Status != keygen.StatusSuccess {
			s += spinner.View(m.spinner)
		}
		s += keygen.View(m.keygen)
	case linkInit:
		s += preamble
		s += spinner.View(m.spinner) + " Generating link..."
	case linkTokenCreated:
		s += preamble
		s += fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			common.Wrap("To link, run the following command on your other machine:"),
			common.Code("charm link "+m.token),
			common.HelpView("To cancel, press escape"),
		)
	case linkRequested:
		var d []string
		s += preamble
		s += "Link request from:\n\n"
		d = append(d, []string{"IP", m.linkRequest.requestAddr}...)
		if len(m.linkRequest.pubKey) > 50 {
			d = append(d, []string{"Key", m.linkRequest.pubKey[0:50] + "..."}...)
		}
		s += common.KeyValueView(d...)
		s += "\n\nLink this device?\n\n"
		s += fmt.Sprintf(
			"%s %s",
			common.YesButtonView(m.buttonIndex == 0),
			common.NoButtonView(m.buttonIndex == 1),
		)
	case linkError:
		s += preamble
		s += "Uh oh: " + m.err.Error()
	case linkSuccess:
		s += common.Keyword("Linked!")
		if m.alreadyLinked {
			s += " This key is already linked, btw."
		}
		if m.standalone {
			s += "\n"
		} else {
			s = preamble + s + common.HelpView("\n\nPress any key to exit...")
		}
	case linkRequestDenied:
		s += "Link request " + common.Keyword("denied") + "."
		if m.standalone {
			s += "\n"
		} else {
			s = preamble + s + common.HelpView("\n\nPress any key to exit...")
		}
	case linkTimedOut:
		s += preamble
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
		s = fmt.Sprintf("\n%s\n", indent.String(s, 2))
	}
	return s
}

// COMMANDS

// InitLinkGen runs the necessary commands for starting the link generation
// process.
func InitLinkGen(m Model) tea.Cmd {
	return tea.Batch(append(HandleLinkRequest(m), spinner.Tick(m.spinner))...)
}

// HandleLinkRequest returns a bunch of blocking commands that resolve on link
// request states. As a Tea command, this should be treated as batch:
//
//     tea.Batch(HandleLinkRequest(model)...)
//
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
