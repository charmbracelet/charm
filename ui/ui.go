package ui

import (
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/info"
	"github.com/charmbracelet/charm/ui/menu"
	"github.com/charmbracelet/charm/ui/username"
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

	info     info.Model
	menu     menu.Model
	username username.Model
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

// SubMap applies a given model to a subscription
// NOTE: if this makes sense, which it likely does, it should be moved to Tea
// core
func SubMap(sub tea.Sub, model tea.Model) tea.Sub {
	return func(_ tea.Model) tea.Msg {
		return sub(model)
	}
}

func initialize(cc *charm.Client) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		s := spinner.NewModel()
		s.Type = spinner.Dot
		s.ForegroundColor = "244"

		m := Model{
			cc:       cc,
			state:    fetching,
			info:     info.NewModel(cc),
			menu:     menu.NewModel(),
			username: username.NewModel(cc),
		}

		return m, CmdMap(info.GetBio, m.info)
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
			if m.menu.Choice != menu.SetUsername {
				m.state = quitting
				return m, tea.Quit
			}
			return m, nil
		case "ctrl+c":
			m.state = quitting
			return m, tea.Quit
		default:
			return updateChilden(msg, m), nil
		}

	case info.GotBioMsg:
		m.state = ready
		m.info, _ = info.Update(msg, m.info)
		return m, nil

	default:
		return updateChilden(msg, m), nil

	}
}

func updateChilden(msg tea.Msg, m Model) Model {
	switch m.state {
	case fetching:
		m.info, _ = info.Update(msg, m.info)
	}

	switch m.menu.Choice {
	case menu.SetUsername:
		m.username, _ = username.Update(msg, m.username)
	default:
		m.menu, _ = menu.Update(msg, m.menu)
	}

	return m
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
		switch m.menu.Choice {
		case menu.SetUsername:
			s += username.View(m.username)
		default:
			s += info.View(m.info)
			s += "\n\n" + menu.View(m.menu)
		}
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

func subscriptions(model tea.Model) tea.Subs {
	m, ok := model.(Model)
	if !ok {
		// TODO: how can we handle this more gracefully?
		return nil
	}

	subs := tea.Subs{}

	switch m.state {
	case fetching:
		subs = AppendSubs(info.Subscriptions(m.info), subs)
	case ready:
		switch m.menu.Choice {
		case menu.SetUsername:
			subs["username-input-blink"] = username.Blink(m.username)
		}
	}

	return subs
}

// AppendSubs merges two groups of subs. Node that subs with idential key names
// will replace existing subs with the same name.
//
// TODO: Move this into Tea core
// TODO: Warn on sub name conflicts and maybe cancel current subs before adding
// new ones
func AppendSubs(newSubs tea.Subs, currentSubs tea.Subs) tea.Subs {
	if len(newSubs) == 0 {
		return currentSubs
	}
	if len(currentSubs) == 0 {
		return newSubs
	}
	for k, v := range newSubs {
		currentSubs[k] = v
	}
	return currentSubs
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
