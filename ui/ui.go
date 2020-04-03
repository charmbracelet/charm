package ui

import (
	"errors"
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/info"
	"github.com/charmbracelet/charm/ui/link"
	"github.com/charmbracelet/charm/ui/username"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	te "github.com/muesli/termenv"
)

const padding = 2

var (
	color       = te.ColorProfile().Color
	cream       = "#FFFDF5"
	purpleBg    = "#5A56E0"
	purpleFg    = "#7571F9"
	fuschia     = "#EE6FF8"
	yellowGreen = "#ECFD65"
)

// NewProgram returns a new tea program
func NewProgram(cc *charm.Client) *tea.Program {
	return tea.NewProgram(initialize(cc), update, view, subscriptions)
}

// state is used to indicate a high level application state
type state int

const (
	fetching state = iota
	ready
	linking
	setUsername
	quitting
)

// menuChoice represents a chosen menu item
type menuChoice int

const (
	copyCharmIDChoice menuChoice = iota
	linkChoice
	setUsernameChoice
	exitChoice
	unsetChoice // set when no choice has been made
)

// menu text corresponding to menu choices. these are presented to the user
var menuChoices = map[menuChoice]string{
	linkChoice:        "Link a machine",
	copyCharmIDChoice: "Copy Charm ID",
	setUsernameChoice: "Set Username",
	exitChoice:        "Exit",
}

// MSG

type copiedCharmIDMsg struct{}

type copyCharmIDErrMsg struct{ error }

// MODEL

// Model holds the state for this program
type Model struct {
	cc            *charm.Client
	user          *charm.User
	err           error
	statusMessage string
	state         state
	menuIndex     int
	menuChoice    menuChoice

	info     info.Model
	link     link.Model
	username username.Model
}

// INIT

func initialize(cc *charm.Client) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		s := spinner.NewModel()
		s.Type = spinner.Dot
		s.ForegroundColor = "244"
		m := Model{
			cc:            cc,
			user:          nil,
			err:           nil,
			statusMessage: "",
			state:         fetching,
			menuIndex:     0,
			menuChoice:    unsetChoice,
			info:          info.NewModel(cc),
			link:          link.NewModel(cc),
			username:      username.NewModel(cc),
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

	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	switch msg := msg.(type) {

	case tea.KeyMsg:

		if m.state == ready { // Process keys for the menu

			switch msg.String() {

			// Quit
			case "q":
				fallthrough
			case "esc":
				fallthrough
			case "ctrl+c":
				m.state = quitting
				return m, tea.Quit

			// Prev menu item
			case "up":
				fallthrough
			case "k":
				m.menuIndex--
				if m.menuIndex < 0 {
					m.menuIndex = len(menuChoices) - 1
				}

			// Select menu item
			case "enter":
				m.menuChoice = menuChoice(m.menuIndex)

			// Next menu item
			case "down":
				fallthrough
			case "j":
				m.menuIndex++
				if m.menuIndex >= len(menuChoices) {
					m.menuIndex = 0
				}
			}
		}

	case copiedCharmIDMsg:
		m.statusMessage = "Copied Charm ID!"
		return m, nil

	case copyCharmIDErrMsg:
		m.err = msg

	case info.GotBioMsg:
		m.state = ready
		m.info, _ = info.Update(msg, m.info)
		m.user = m.info.User

	case username.NameSetMsg:
		m.state = ready
		m.username = username.NewModel(m.cc) // reset the state
		m.info.User.Name = string(msg)

	}

	m.statusMessage = ""
	m, cmd = updateChilden(msg, m)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func updateChilden(msg tea.Msg, m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {
	case fetching:
		m.info, _ = info.Update(msg, m.info)
		if m.info.Quit {
			m.state = quitting
			m.err = m.info.Err
			return m, tea.Quit
		}
		return m, nil
	case linking:
		m.link, _ = link.Update(msg, m.link)
		if m.link.Exit {
			m.link = link.NewModel(m.cc) // reset the state
			m.state = ready
		} else if m.link.Quit {
			m.state = quitting
			return m, tea.Quit
		}
	case setUsername:
		m.username, cmd = username.Update(msg, m.username)
		if m.username.Done {
			m.username = username.NewModel(m.cc) // reset the state
			m.state = ready
		} else if m.username.Quit {
			m.state = quitting
			return m, tea.Quit
		}
	}

	switch m.menuChoice {
	case linkChoice:
		m.state = linking
		m.menuChoice = unsetChoice
		cmd = tea.CmdMap(link.GenerateLink, m.link)
	case setUsernameChoice:
		m.state = setUsername
		m.menuChoice = unsetChoice
	case copyCharmIDChoice:
		cmd = copyCharmIDCmd
		m.menuChoice = unsetChoice
	case exitChoice:
		m.state = quitting
		cmd = tea.Quit
	}

	return m, cmd
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
		s += info.View(m.info)
		s += "\n\n" + menuView(m.menuIndex)
		s += footerView(m)
	case linking:
		s += link.View(m.link)
	case setUsername:
		s += username.View(m.username)
	case quitting:
		s += quitView(m)
	}

	return indent.String(s, padding)
}

func charmLogoView() string {
	title := te.String(" Charm ").Foreground(color(cream)).Background(color(purpleBg)).String()
	return "\n" + title + "\n\n"
}

func menuView(currentIndex int) string {
	var s string
	for i := 0; i < len(menuChoices); i++ {
		e := "  "
		if i == currentIndex {
			e = te.String("> ").Foreground(color(purpleBg)).String()
			e += te.String(menuChoices[menuChoice(i)]).Foreground(color(purpleFg)).String()
		} else {
			e += menuChoices[menuChoice(i)]
		}
		if i < len(menuChoices)-1 {
			e += "\n"
		}
		s += e
	}
	return s
}

func quitView(m Model) string {
	if m.err != nil {
		return fmt.Sprintf("Uh oh, there’s been an error: %v\n", m.err)
	}
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
	s := "j/k, ↑/↓: choose • enter: select"
	return te.String(s).Foreground(color("241")).String()
}

func statusMessageView(s string) string {
	return te.String(s).Foreground(color(fuschia)).String()
}

func errorView(err error) string {
	head := te.String("Error: ").Foreground(color("203")).String()
	msg := te.String(
		wordwrap.String(err.Error(), 50),
	).Foreground(color("241")).String()
	return indent.String(head+msg, 2)
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
		subs["info-spinner-tick"] = info.Tick(m.info)
	case setUsername:
		subs["username-input-blink"] = username.Blink(m.username)
		subs["username-spinner-tick"] = username.Spin(m.username)
	}

	return subs
}

// COMMANDS

// copyCharmIDCmd copies the Charm ID to the clipboard
func copyCharmIDCmd(model tea.Model) tea.Msg {
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
