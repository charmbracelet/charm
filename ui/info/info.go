package info

// Fetch a user's basic Charm account info

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
)

// GotBioMsg is sent when we've successfully fetched the user's bio. It
// contains the user's profile data.
type GotBioMsg *charm.User

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

// Model stores the state of the info user interface.
type Model struct {
	Quit bool // signals it's time to exit the whole application
	Err  error
	User *charm.User
	cc   *charm.Client
}

// NewModel returns a new Model in its initial state.
func NewModel(cc *charm.Client) Model {
	return Model{
		Quit: false,
		User: nil,
		cc:   cc,
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
func View(m Model) string {
	if m.Err != nil {
		return "error: " + m.Err.Error()
	} else if m.User == nil {
		return " Authenticating..."
	}
	return bioView(m.User)
}

func bioView(u *charm.User) string {
	var username string
	if u.Name != "" {
		username = u.Name
	} else {
		username = common.Subtle("(none set)")
	}
	return common.KeyValueView(
		"Username", username,
		"Joined", u.CreatedAt.Format("02 Jan 2006"),
	)
}

// GetBio fetches the authenticated user's bio.
func GetBio(cc *charm.Client) tea.Cmd {
	return func() tea.Msg {
		user, err := cc.Bio()
		if err != nil {
			return errMsg{err}
		}

		return GotBioMsg(user)
	}
}
