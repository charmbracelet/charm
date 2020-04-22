package linkgen

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
	"github.com/muesli/reflow/indent"
)

type status int

const (
	linkInit status = iota
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
type errMsg error

// NewProgram is a simple wrapper for tea.NewProgram
func NewProgram(cc *charm.Client) *tea.Program {
	return tea.NewProgram(Init(cc), Update, View, Subscriptions)
}

// Model is the Tea model for the link initiator program
type Model struct {
	lh            *linkHandler
	standalone    bool // true if this is running as a stadalone Tea program
	Quit          bool // indicates the user wants to exit the whole program
	Exit          bool // indicates the user wants to exit this mini-app
	err           error
	status        status
	alreadyLinked bool
	token         string
	linkRequest   linkRequest
	cc            *charm.Client
	buttonIndex   int // focused state of ok/cancel buttons
	spinner       spinner.Model
}

// acceptRequest rejects the current linking request
func (m Model) acceptRequest() (Model, tea.Cmd) {
	m.lh.response <- true
	return m, nil
}

// rejectRequset rejects the current linking request
func (m Model) rejectRequest() (Model, tea.Cmd) {
	m.lh.response <- false
	m.status = linkRequestDenied
	if m.standalone {
		return m, tea.Quit
	}
	return m, nil
}

func NewModel(cc *charm.Client) Model {
	lh := &linkHandler{
		err:      make(chan error),
		token:    make(chan string),
		request:  make(chan linkRequest),
		response: make(chan bool),
		success:  make(chan bool),
		timeout:  make(chan struct{}),
	}
	s := spinner.NewModel()
	s.Type = spinner.Dot
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
		cc:            cc,
		buttonIndex:   0,
		spinner:       s,
	}
}

// Init is a Tea program's initialization function
func Init(cc *charm.Client) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		m := NewModel(cc)
		m.standalone = true
		return m, tea.Batch(HandleLinkRequest(m)...)
	}
}

// Update is the Tea update loop
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
		case "q":
			fallthrough
		case "esc":
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
				case "j":
					fallthrough
				case "h":
					fallthrough
				case "right":
					fallthrough
				case "tab":
					m.buttonIndex++
					if m.buttonIndex > 1 {
						m.buttonIndex = 0
					}
				case "k":
					fallthrough
				case "l":
					fallthrough
				case "left":
					fallthrough
				case "shift+tab":
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

			case linkSuccess:
				fallthrough
			case linkRequestDenied:
				fallthrough
			case linkTimedOut:
				// Any key exits
				m.Exit = true
				return m, nil

			}
		}

	case errMsg:
		m.status = linkError
		m.err = msg
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
		m.spinner, _ = spinner.Update(msg, m.spinner)
		return m, nil
	}

	return m, nil
}

// View renders the UI
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
	case linkInit:
		s += preamble
		s += spinner.View(m.spinner) + " Generating link..."
	case linkTokenCreated:
		s += preamble
		s += fmt.Sprintf(
			"%s\n\n%s%s",
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
			s += " This account is already linked, btw."
		}
		if m.standalone {
			s += "\n"
		} else {
			s = preamble + s + common.HelpView("Press any key to exit...")
		}
	case linkRequestDenied:
		s += "Link request " + common.Keyword("denied") + "."
		if m.standalone {
			s += "\n"
		} else {
			s = preamble + s + common.HelpView("Press any key to exit...")
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
		s = fmt.Sprintf("\n%s", indent.String(s, 2))
	}
	return s
}

// SUBSCRIPTIONS

// Subscriptions returns Tea subscriptions when using this componenent as a
// standalone program.
func Subscriptions(model tea.Model) tea.Subs {
	m, ok := model.(Model)
	if !ok {
		return nil
	}
	return tea.Subs{
		"link-spinner-tick": Spin(m),
	}
}

// Spin wraps the spinner components's subscription. This should be integrated
// when this component is used as part of another program.
func Spin(model tea.Model) tea.Sub {
	m, ok := model.(Model)
	if !ok {
		return nil
	}

	if m.status != linkInit {
		return nil
	}
	return tea.SubMap(spinner.Sub, m.spinner)
}

// COMMANDS

// HandleLinkRequest returns a bunch of blocking commands that resolve on link
// request states. As a Tea command, this should be treated as batch:
//
//     tea.Batch(HandleLinkRequest(model)...)
//
func HandleLinkRequest(m Model) []tea.Cmd {

	go func() {
		m.cc.RenewSession()
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
			return errMsg(err)
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

// handleLinkError responds when a linking error is reported
func handleLinkError(lh *linkHandler) tea.Cmd {
	return func() tea.Msg {
		return errMsg(<-lh.err)
	}
}
