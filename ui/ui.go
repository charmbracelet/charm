package ui

import (
	"log"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/info"
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

type state int

const (
	fetching state = iota
	ready
	quitting
)

// MODEL

type Model struct {
	cc    *charm.Client
	user  *charm.User
	err   error
	state state

	info info.Model
	menu menu.Model
}

// INIT

// CmdMap applies a given model to a command
// NOTE: if this makes sense, which it likely does, it should be moved to Tea
// core
func CmdMap(cmd tea.Cmd, model tea.Model) tea.Cmd {
	return func(_ tea.Model) tea.Msg {
		return cmd(model)
	}
}

func initialize(cc *charm.Client) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		s := spinner.NewModel()
		s.Type = spinner.Dot
		s.ForegroundColor = "244"

		m := Model{
			cc:    cc,
			state: fetching,
			info:  info.NewModel(cc),
			menu:  menu.NewModel(),
		}

		cmd := CmdMap(info.GetBio, m.info)

		return m, cmd
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

	case info.GotBioMsg:
		var cmd tea.Cmd
		m.state = ready
		m.info, cmd = info.Update(msg, m.info)
		return m, cmd

	default:
		m.info, _ = info.Update(msg, m.info)
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
		s += info.View(m.info)
	case ready:
		s += info.View(m.info)
		s += menu.View(m.menu)
	case quitting:
		s += quitView()
	}

	return indent.String(s, padding)
}

func charmLogoView() string {
	return "\n" + fgBg(" Charm ", cream, purpleBg).Bold().String() + "\n\n"
}

func quitView() string {
	return "Thanks for using Charm!\n"
}

func errorView(err error) string {
	return indent.String("\n"+fgBg("ERROR", "230", "203").String()+" "+err.Error(), padding)
}

// SUBSCRIPTIONS

func subscriptions(model tea.Model) (subs tea.Subs) {
	m, ok := model.(Model)
	if !ok {
		// TODO: is there a more graceful way to handle this?
		log.Fatal("could not corerce model in main subscriptions function")
	}

	// NOTE: Eventually, we'll need to append sub maps together here. Something
	// like that should probably go into Tea core.
	subs = info.Subscriptions(m.info)
	return subs
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
