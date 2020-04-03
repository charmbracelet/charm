package link

import (
	"fmt"
	"log"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
	"github.com/muesli/reflow/wordwrap"
)

type linkTokenCreatedMsg string

type errMsg struct {
	error
}

func (err errMsg) String() string {
	return err.Error()
}

type Model struct {
	Quit   bool // indicates the user wants to exit the whole program
	Exit   bool // indicates the user wants to exit this mini-app
	err    error
	status charm.LinkStatus
	token  string
	cc     *charm.Client
}

func NewModel(cc *charm.Client) Model {
	return Model{
		Quit:   false,
		Exit:   false,
		err:    nil,
		status: charm.LinkStatusInit,
		token:  "",
		cc:     cc,
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

// Initialize returns a new model and initial command
func Initialize(cc *charm.Client) (Model, tea.Cmd) {
	return NewModel(cc), GenerateLink
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
		}

	case errMsg:
		m.status = charm.LinkStatusError
		m.err = msg
		return m, nil

	case linkTokenCreatedMsg:
		m.status = charm.LinkStatusTokenCreated
		m.token = string(msg)
		return m, nil

	}

	return m, nil
}

// View renders the UI
func View(m Model) string {
	s := wordwrap.String("You can link the SSH keys on another machine to your Charm account so both machines have access to your stuff. Rest assured that you can also unlink keys at any time.\n\nReady to go?", 50)
	switch m.status {
	case charm.LinkStatusTokenCreated:
		s += "\n\ncharm link " + m.token
	case charm.LinkStatusError:
		s += "Uh oh: " + m.err.Error()
	}
	return s
}

// COMMANDS

// GenerateLink starts the linking process by creating a token
func GenerateLink(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.ModelAssertionErr
	}

	lh := &linkHandler{token: make(chan string)}
	errChan := make(chan error)

	go func() {
		m.cc.RenewSession()
		if err := m.cc.LinkGen(lh); err != nil {
			errChan <- err
			return
		}
	}()

	select {
	case err := <-errChan:
		return errMsg{err}
	case token := <-lh.token:
		return linkTokenCreatedMsg(token)
	}
}

// LINK HANDLER

// linkHandler implements the charm.LinkHandler interface
type linkHandler struct {
	token chan string
}

func (lh *linkHandler) TokenCreated(l *charm.Link) {
	log.Printf("To link a machine, run: \n\n> charm link %s\n", l.Token)
	lh.token <- l.Token
}

func (lh *linkHandler) TokenSent(l *charm.Link) {
	log.Println("Linking...")
}

func (lh *linkHandler) ValidToken(l *charm.Link) {
	log.Println("Valid token")
}

func (lh *linkHandler) InvalidToken(l *charm.Link) {
	log.Println("That token looks invalid.")
}

func (lh *linkHandler) Request(l *charm.Link) bool {
	log.Printf("Does this look right? (yes/no)\n\n%s\nIP: %s\n", l.RequestPubKey, l.RequestAddr)
	//if strings.ToLower(conf) == "yes\n" {
	//return true
	//}
	return false
}

func (lh *linkHandler) RequestDenied(l *charm.Link) {
	log.Println("Not Linked :(")
}

func (lh *linkHandler) SameAccount(l *charm.Link) {
	fmt.Println("Linked! You already linked this key btw.")
}

func (lh *linkHandler) Success(l *charm.Link) {
	log.Println("Linked!")
}

func (lh *linkHandler) Timeout(l *charm.Link) {
	log.Println("Timed out. Sorry.")
}

func (lh *linkHandler) Error(l *charm.Link) {
	log.Println("Error, something's wrong.")
}
