package link

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
)

type linkTokenCreatedMsg string
type linkRequestMsg linkRequest
type linkSuccessMsg bool // true if this account's already been linked
type linkTimeoutMsg struct{}

type errMsg struct {
	error
}

func (err errMsg) String() string {
	return err.Error()
}

type Model struct {
	lh            *linkHandler
	Quit          bool // indicates the user wants to exit the whole program
	Exit          bool // indicates the user wants to exit this mini-app
	err           error
	status        charm.LinkStatus
	alreadyLinked bool
	token         string
	linkRequest   linkRequest
	cc            *charm.Client
	buttonIndex   int // focused state of ok/cancel buttons
	spinner       spinner.Model
}

// acceptRequest rejects the current linking request
func (m *Model) acceptRequest() {
	m.lh.response <- true
}

// rejectRequset rejects the current linking request
func (m *Model) rejectRequest() {
	m.lh.response <- false
	m.status = charm.LinkStatusRequestDenied
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
		Quit:          false,
		Exit:          false,
		err:           nil,
		status:        charm.LinkStatusInit,
		alreadyLinked: false,
		token:         "",
		linkRequest:   linkRequest{},
		cc:            cc,
		buttonIndex:   0,
		spinner:       s,
	}
}

func (m *Model) CancelRequest() {
	if m.cc == nil {
		return
	}
	if err := m.cc.CloseSession(); err != nil {
		m.err = err
	}
}

// Update is the Tea update loop
func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.String() {

		// General keybindings
		case "ctrl+c":
			m.CancelRequest()
			m.Quit = true
			return m, nil
		case "q":
			fallthrough
		case "esc":
			m.CancelRequest()
			m.Exit = true
			return m, nil

		// State-specific keybindings
		default:
			switch m.status {

			case charm.LinkStatusRequested:
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
						m.acceptRequest()
					} else {
						m.rejectRequest()
					}
				case "y":
					m.acceptRequest()
				case "n":
					m.rejectRequest()
				}
				return m, nil

			case charm.LinkStatusSuccess:
				fallthrough
			case charm.LinkStatusRequestDenied:
			case charm.LinkStatusTimedOut:
				// Any key exits
				m.Exit = true
				return m, nil

			}
		}

	case errMsg:
		m.status = charm.LinkStatusError
		m.err = msg
		return m, nil

	case linkTokenCreatedMsg:
		m.status = charm.LinkStatusTokenCreated
		m.token = string(msg)
		return m, nil

	case linkRequestMsg:
		m.status = charm.LinkStatusRequested
		m.linkRequest = linkRequest(msg)
		return m, nil

	case linkSuccessMsg:
		m.status = charm.LinkStatusSuccess
		m.alreadyLinked = bool(msg)
		return m, nil

	case linkTimeoutMsg:
		m.status = charm.LinkStatusTimedOut
		return m, nil

	case spinner.TickMsg:
		m.spinner, _ = spinner.Update(msg, m.spinner)
		return m, nil
	}

	switch m.status {
	case charm.LinkStatusRequested:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch strings.ToLower(msg.String()) {
			case "y":
				// Accept request
				m.lh.response <- true
				return m, nil
			case "n":
				// Reject request
				m.status = charm.LinkStatusRequestDenied
				m.lh.response <- false
				return m, nil
			}
		}
	}

	return m, nil
}

// View renders the UI
func View(m Model) string {
	s := common.Wrap(fmt.Sprintf(
		"You can %s the SSH keys on another machine to your Charm account so both machines have access to your stuff. You can unlink keys at any time.\n\n",
		common.Keyword("link"),
	))
	switch m.status {
	case charm.LinkStatusInit:
		s += spinner.View(m.spinner) + " Generating link..."
	case charm.LinkStatusTokenCreated:
		s += fmt.Sprintf(
			"%s\n\n%s%s",
			common.Wrap("To link, run the following command on your other machine:"),
			common.Code("charm link "+m.token),
			common.HelpView("To cancel, press escape"),
		)
	case charm.LinkStatusRequested:
		var d []string
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
	case charm.LinkStatusError:
		s += "Uh oh: " + m.err.Error()
	case charm.LinkStatusSuccess:
		also := ""
		if m.alreadyLinked {
			also = " This account is already linked, btw."
		}
		s += common.Keyword("Linked!") + also + common.HelpView("Press any key to exit...")
	case charm.LinkStatusRequestDenied:
		s += "Link request " + common.Keyword("denied") + common.HelpView("Press any key to exit...")
	case charm.LinkStatusTimedOut:
		s += "Link request timed out." + common.HelpView("Press any key to exit...")
	}
	return s
}

// SUBSCRIPTIONS

func Spin(model tea.Model) tea.Sub {
	m, ok := model.(Model)
	if !ok {
		return nil
	}

	if m.status != charm.LinkStatusInit {
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
func HandleLinkRequest(model tea.Model) []tea.Cmd {
	m, ok := model.(Model)
	if !ok {
		return []tea.Cmd{func(_ tea.Model) tea.Msg {
			return tea.ModelAssertionErr
		}}
	}

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
	return func(_ tea.Model) tea.Msg {
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
	return func(_ tea.Model) tea.Msg {
		return linkRequestMsg(<-lh.request)
	}
}

// handleLinkSuccess waits for data in the link success channel.
func handleLinkSuccess(lh *linkHandler) tea.Cmd {
	return func(_ tea.Model) tea.Msg {
		return linkSuccessMsg(<-lh.success)
	}
}

// handleLinkTimeout waits for a timeout in the linking process.
func handleLinkTimeout(lh *linkHandler) tea.Cmd {
	return func(_ tea.Model) tea.Msg {
		<-lh.timeout
		return linkTimeoutMsg{}
	}
}

// handleLinkError responds when a linking error is reported
func handleLinkError(lh *linkHandler) tea.Cmd {
	return func(_ tea.Model) tea.Msg {
		return errMsg{<-lh.err}
	}
}

// LINK HANDLING

// linkRequest carries metadata pertaining to a link request
type linkRequest struct {
	pubKey      string
	requestAddr string
}

// linkHandler implements the charm.LinkHandler interface
type linkHandler struct {
	err      chan error
	token    chan string
	request  chan linkRequest
	response chan bool
	success  chan bool
	timeout  chan struct{}
}

func (lh *linkHandler) TokenCreated(l *charm.Link) {
	lh.token <- l.Token
}

func (lh *linkHandler) TokenSent(l *charm.Link) {}

func (lh *linkHandler) ValidToken(l *charm.Link) {}

func (lh *linkHandler) InvalidToken(l *charm.Link) {}

// Request handles link approvals. The remote machine sends an approval request,
// which we send to the Tea UI as a message. The Tea application then sends a
// response to the link handler's response channel with a command.
func (lh *linkHandler) Request(l *charm.Link) bool {
	lh.request <- linkRequest{l.RequestPubKey, l.RequestAddr}
	return <-lh.response
}

func (lh *linkHandler) RequestDenied(l *charm.Link) {}

// Successful link, but this account has already been linked
func (lh *linkHandler) SameAccount(l *charm.Link) {
	lh.success <- true
}

func (lh *linkHandler) Success(l *charm.Link) {
	lh.success <- false
}

func (lh *linkHandler) Timeout(l *charm.Link) {
	lh.timeout <- struct{}{}
}

func (lh *linkHandler) Error(l *charm.Link) {
	lh.err <- errors.New("there’s been an error; please try again")
}
