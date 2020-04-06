package link

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
	"github.com/muesli/reflow/wordwrap"
)

type linkTokenCreatedMsg string
type linkRequestMsg linkRequest
type linkSuccessMsg struct{}

type errMsg struct {
	error
}

func (err errMsg) String() string {
	return err.Error()
}

type Model struct {
	lh          *linkHandler
	Quit        bool // indicates the user wants to exit the whole program
	Exit        bool // indicates the user wants to exit this mini-app
	err         error
	status      charm.LinkStatus
	token       string
	linkRequest linkRequest
	cc          *charm.Client
}

func NewModel(cc *charm.Client) Model {
	lh := &linkHandler{
		err:      make(chan error),
		token:    make(chan string),
		request:  make(chan linkRequest),
		response: make(chan bool),
		success:  make(chan struct{}),
	}
	return Model{
		lh:          lh,
		Quit:        false,
		Exit:        false,
		err:         nil,
		status:      charm.LinkStatusInit,
		token:       "",
		linkRequest: linkRequest{},
		cc:          cc,
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
		default:
			if m.status == charm.LinkStatusSuccess {
				// After a successful connection any key returns to the menu.
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
				m.lh.response <- false
				return m, nil
			}
		}
	}

	return m, nil
}

// View renders the UI
func View(m Model) string {
	s := wordwrap.String("You can link the SSH keys on another machine to your Charm account so both machines have access to your stuff. Rest assured that you can also unlink keys at any time.\n\n", 50)
	switch m.status {
	case charm.LinkStatusInit:
		s += "Generating link..."
	case charm.LinkStatusTokenCreated:
		s += wordwrap.String("To link, run the following command on your other machine:", 50)
		s += "\n\ncharm link " + m.token
	case charm.LinkStatusRequested:
		s += "Link request from:\n\n"
		s += fmt.Sprintf("IP: %s\n", m.linkRequest.requestAddr)
		if len(m.linkRequest.pubKey) > 50 {
			s += fmt.Sprintf("Key: %s...", m.linkRequest.pubKey[0:50])
		}
		s += "\n\nLink your account to this device? y/n"
	case charm.LinkStatusError:
		s += "Uh oh: " + m.err.Error()
	case charm.LinkStatusSuccess:
		s += "Linked!\n\nPress any key to exit..."
	}
	return s
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
		<-lh.success
		return linkSuccessMsg{}
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
	success  chan struct{}
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

func (lh *linkHandler) SameAccount(l *charm.Link) {
	fmt.Println("Linked! You already linked this key btw.")
}

func (lh *linkHandler) Success(l *charm.Link) {
	lh.success <- struct{}{}
}

func (lh *linkHandler) Timeout(l *charm.Link) {
	log.Println("Timed out. Sorry.")
}

func (lh *linkHandler) Error(l *charm.Link) {
	log.Println("Error, something's wrong.")
}
