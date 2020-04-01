package ui

import (
	"errors"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/info"
	"github.com/charmbracelet/charm/ui/menu"
	"github.com/charmbracelet/charm/ui/username"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const padding = 2

var (
	color    = te.ColorProfile().Color
	purpleBg = "#5A56E0"
	purpleFg = "#7571F9"
	cream    = "#FFFDF5"
)

// NewProgram returns a new tea program
func NewProgram(cc *charm.Client) *tea.Program {
	return tea.NewProgram(initialize(cc), update, view, subscriptions)
}

type state int

const (
	fetching state = iota
	ready
	quitting
)

// MSG

type copiedCharmIDMsg struct{}

type copyCharmIDErrMsg struct{ error }

// MODEL

type Model struct {
	cc            *charm.Client
	user          *charm.User
	err           error
	statusMessage string
	state         state

	info     info.Model
	menu     menu.Model
	username username.Model
}

// INIT

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

		return m, tea.CmdMap(info.GetBio, m.info)
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
			if m.menu.Choice == menu.SetUsername {
				var cmd tea.Cmd
				m.username, cmd = username.Update(msg, m.username)
				return m, cmd
			}
			m.state = quitting
			return m, tea.Quit
		case "ctrl+c":
			m.state = quitting
			return m, tea.Quit
		default:
			m.statusMessage = ""
			return updateChilden(msg, m)
		}

	case copiedCharmIDMsg:
		m.statusMessage = "Copied Charm ID"
		return m, nil

	case copyCharmIDErrMsg:
		m.err = msg
		return m, nil

	case info.GotBioMsg:
		m.state = ready
		m.info, _ = info.Update(msg, m.info)
		m.user = m.info.User
		return m, nil

	case username.ExitMsg:
		m.menu.Choice = menu.Unset
		return m, nil

	default:
		m.statusMessage = ""
		return updateChilden(msg, m)

	}
}

func updateChilden(msg tea.Msg, m Model) (Model, tea.Cmd) {
	if m.state == fetching {
		m.info, _ = info.Update(msg, m.info)
		return m, nil
	}

	prevChoice := m.menu.Choice
	m.menu, _ = menu.Update(msg, m.menu)

	switch m.menu.Choice {
	case menu.SetUsername:
		// If the user is navigating to the username component, don't let user
		// input in this update pass through, otherwise the enter key we're
		// using to navigate in will also submit the form
		if prevChoice == menu.SetUsername {
			var cmd tea.Cmd
			m.username, cmd = username.Update(msg, m.username)
			return m, cmd
		}
		return m, nil
	case menu.CopyCharmID:
		m.menu.Choice = menu.Unset
		return m, copyCharmID
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
			s += footerView(m)
		}
	case quitting:
		s += quitView()
	}

	return indent.String(s, padding)
}

func charmLogoView() string {
	title := te.String(" Charm ").Foreground(color(cream)).Background(color(purpleBg)).String()
	return "\n" + title + "\n\n"
}

func quitView() string {
	return "Thanks for using Charm!\n"
}

func footerView(m Model) string {
	s := "\n\n"
	if m.err != nil {
		return s + errorView(m.err)
	}
	if m.statusMessage != "" {
		return s + statusMessageView(m.statusMessage)
	}
	return s + helpView()
}

func helpView() string {
	s := "j/k, up/down: choose • enter: select • q: exit"
	return te.String(s).Foreground(color("241")).String()
}

func statusMessageView(s string) string {
	return te.String(s).Foreground(color(purpleFg)).String()
}

func errorView(err error) string {
	head := te.String(" ERROR ").Foreground(color("230")).Background(color("203")).String()
	return indent.String(head+" "+err.Error(), padding)
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

// COMMANDS

func copyCharmID(model tea.Model) tea.Msg {
	m, ok := model.(Model)
	if !ok {
		return tea.ModelAssertionErr
	}
	if m.user == nil {
		return copyCharmIDErrMsg{errors.New("we don't have any user info")}
	}

	if err := clipboard.WriteAll(m.user.CharmID); err != nil {
		return copyCharmIDErrMsg{err}
	}
	return copiedCharmIDMsg{}
}
