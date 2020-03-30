package ui

import (
	"log"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/menu"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/termenv"
)

const padding = 2

var (
	color    = termenv.ColorProfile().Color
	purpleBg = "#5A56E0"
	purpleFg = "#7571F9"
	cream    = "#FFFDF5"
)

// New Program returns a new tea program
func NewProgram(cc *charm.Client) *tea.Program {
	return tea.NewProgram(initialize(cc), update, view, subscriptions)
}

type State int

const (
	fetching State = iota
	fetched
	quitting
)

// MSG

type GotBioMsg *charm.User

// MODEL

type Model struct {
	client  *charm.Client
	user    *charm.User
	spinner spinner.Model
	menu    menu.Model
	err     error
	state   State
}

// INIT

func initialize(cc *charm.Client) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		s := spinner.NewModel()
		s.Type = spinner.Dot
		s.ForegroundColor = "244"

		m := Model{
			client:  cc,
			spinner: s,
			menu:    menu.Model{},
			state:   fetching,
		}
		return m, getBio
	}
}

// UPDATE

func update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m, ok := model.(Model)
	if !ok {
		m.err = tea.ModelAssertionErr
		return m, nil
	}

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "q":
			fallthrough
		case "ctrl+c":
			m.state = quitting
			return m, tea.Quit

		default:
			m.menu, _ = menu.Update(msg, m.menu)
			return m, nil
		}

	case GotBioMsg:
		m.user = msg
		m.state = fetched
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = spinner.Update(msg, m.spinner)
		return m, cmd

	default:
		return m, nil
	}
}

// VIEW

func view(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		m.err = tea.ModelAssertionErr
	}

	if m.err != nil {
		return errorView(m.err)
	}

	s := charmLogoView()

	switch m.state {
	case fetching:
		s += spinner.View(m.spinner) + " Fetching your information...\n"
	case fetched:
		s += bioView(*m.user)
		s += menu.View(m.menu)
	case quitting:
		s += quitView()
	}

	return indent.String(s, padding)
}

func charmLogoView() string {
	return "\n" + fgBg(" Charm ", cream, purpleBg).Bold().String() + "\n\n"
}

func bioView(u charm.User) string {
	bar := fg("│ ", "241").String()
	username := fg("not set", "214").String()
	if u.Name != "" {
		username = fg(u.Name, purpleFg).String()
	}
	return bar + "Charm ID " + fg(u.CharmID, purpleFg).String() + "\n" +
		bar + "Username " + username
}

func quitView() string {
	return "Thanks for using Charm!\n"
}

func errorView(err error) string {
	return indent.String("\n"+fgBg("ERROR", "230", "203").String()+" "+err.Error(), padding)
}

// SUBSCRIPTIONS

func subscriptions(model tea.Model) tea.Subs {
	m, ok := model.(Model)
	if !ok {
		// TODO: is there a more graceful way to handle this?
		log.Fatal("could not corerce model in main subscriptions function")
	}

	return tea.Subs{
		"tick": func(model tea.Model) tea.Msg {
			return spinner.Sub(m.spinner)
		},
	}

}

// COMMANDS

func getBio(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.NewErrMsgFromErr(tea.ModelAssertionErr)
	}

	user, err := m.client.Bio()
	if err != nil {
		return tea.NewErrMsgFromErr(err)
	}

	return GotBioMsg(user)
}

// HELPERS

func fg(s string, fgColor string) termenv.Style {
	return termenv.String(s).
		Foreground(color(fgColor))
}

func fgBg(s, fgColor, bgColor string) termenv.Style {
	return termenv.String(s).
		Foreground(color(fgColor)).
		Background(color(bgColor))
}
