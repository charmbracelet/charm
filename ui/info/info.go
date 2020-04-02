package info

// Fetch a user's basic Charm account info

import (
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
	te "github.com/muesli/termenv"
)

var (
	color    = te.ColorProfile().Color
	purpleBg = "#5A56E0"
	purpleFg = "#7571F9"
)

// MSG

type GotBioMsg *charm.User

// MODEL

type Model struct {
	Quit    bool // signals it's time to exit the whole application
	User    *charm.User
	err     error
	cc      *charm.Client
	spinner spinner.Model
}

func NewModel(cc *charm.Client) Model {
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = "244"

	return Model{
		Quit:    false,
		User:    nil,
		cc:      cc,
		spinner: s,
	}
}

// UPDATE

func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case GotBioMsg:
		m.User = msg
	case tea.ErrMsg:
		// If there's an error we print the error and exit
		m.err = msg
		m.Quit = true
		return m, nil
	case spinner.TickMsg:
		m.spinner, _ = spinner.Update(msg, m.spinner)
	}
	return m, nil
}

// VIEW

func View(m Model) string {
	if m.err != nil {
		return "error: " + m.err.Error() + "\n"
	} else if m.User == nil {
		return spinner.View(m.spinner) + " Fetching your information...\n"
	}
	return bioView(m.User)
}

func bioView(u *charm.User) string {
	var username string
	bar := te.String("│ ").Foreground(color("241")).String()
	if u.Name != "" {
		username = te.String(u.Name).Foreground(color(purpleFg)).String()
	} else {
		username = te.String("(none set)").Foreground(color("241")).String()
	}
	id := te.String(u.CharmID).Foreground(color(purpleFg)).String()
	return bar + "Charm ID " + id + "\n" +
		bar + "Username " + username
}

// SUBSCRIPTIONS

// Tick just wraps the spinner's subscription
func Tick(model tea.Model) tea.Sub {
	m, ok := model.(Model)
	if !ok {
		// TODO: handle this error properly
		return nil
	} else if m.User != nil {
		return nil
	}

	return tea.SubMap(spinner.Sub, m.spinner)
}

// COMMANDS

func GetBio(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.ModelAssertionErr
	}

	user, err := m.cc.Bio()
	if err != nil {
		return tea.NewErrMsgFromErr(err)
	}

	return GotBioMsg(user)
}
