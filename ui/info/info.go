package info

// Fetch a user's basic Charm account info

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui/common"
)

// GotBioMsg is sent when we've successfully fetched the user's bio. It
// contains the user's profile data.
type GotBioMsg *charm.User

type errMsg struct {
	err error
}

// Error satisfies the error interface.
func (e errMsg) Error() string {
	return e.err.Error()
}

// Model stores the state of the info user interface.
type Model struct {
	Quit   bool // signals it's time to exit the whole application
	Err    error
	User   *charm.User
	cc     *client.Client
	styles common.Styles
}

// NewModel returns a new Model in its initial state.
func NewModel(cc *client.Client) Model {
	return Model{
		Quit:   false,
		User:   nil,
		cc:     cc,
		styles: common.DefaultStyles(),
	}
}

// Update is the Bubble Tea update loop.
func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.Quit = true
			return m, nil
		}
	case GotBioMsg:
		m.User = msg
	case errMsg:
		// If there's an error we print the error and exit
		m.Err = msg
		m.Quit = true
		return m, nil
	}

	return m, cmd
}

// View renders the current view from the model.
func (m Model) View() string {
	if m.Err != nil {
		return "error: " + m.Err.Error()
	} else if m.User == nil {
		return " Authenticating..."
	}
	return m.bioView()
}

func (m Model) bioView() string {
	var username string
	if m.User.Name != "" {
		username = m.User.Name
	} else {
		username = m.styles.Subtle.Render("(none set)")
	}
	return common.KeyValueView(
		"Host", m.cc.Config.Host,
		"Username", username,
		"Joined", m.User.CreatedAt.Format("02 Jan 2006"),
	)
}

// GetBio fetches the authenticated user's bio.
func GetBio(cc *client.Client) tea.Cmd {
	return func() tea.Msg {
		user, err := cc.Bio()
		if err != nil {
			return errMsg{err}
		}

		return GotBioMsg(user)
	}
}
