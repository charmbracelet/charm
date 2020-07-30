package info

// Fetch a user's basic Charm account info

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	te "github.com/muesli/termenv"
)

var (
	color    = te.ColorProfile().Color
	purpleBg = "#5A56E0"
	purpleFg = "#7571F9"
)

// MSG

type GotBioMsg *charm.User

type errMsg error

// MODEL

type Model struct {
	Quit bool // signals it's time to exit the whole application
	Err  error
	User *charm.User
	cc   *charm.Client
}

func NewModel(cc *charm.Client) Model {
	return Model{
		Quit: false,
		User: nil,
		cc:   cc,
	}
}

// UPDATE

func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			fallthrough
		case "esc":
			fallthrough
		case "ctrl+c":
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

// VIEW

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
		username = te.String("(none set)").Foreground(color("241")).String()
	}
	return common.KeyValueView(
		"Username", username,
		"Joined", u.CreatedAt.Format("02 Jan 2006"),
	)
}

// COMMANDS

// GetBio fetches the authenticated user's bio
func GetBio(cc *charm.Client) tea.Cmd {
	return func() tea.Msg {
		user, err := cc.Bio()
		if err != nil {
			return errMsg(err)
		}

		return GotBioMsg(user)
	}
}
