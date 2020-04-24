package ui

import (
	"errors"
	"fmt"
	"log"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/info"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/charm/ui/keys"
	"github.com/charmbracelet/charm/ui/linkgen"
	"github.com/charmbracelet/charm/ui/username"
	"github.com/charmbracelet/tea"
	"github.com/charmbracelet/teaparty/spinner"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const padding = 2

// NewProgram returns a new tea program
func NewProgram(cfg *charm.Config) *tea.Program {
	return tea.NewProgram(initialize(cfg), update, view, subscriptions)
}

// status is used to indicate a high level application state
type status int

const (
	statusInit status = iota
	statusKeygen
	statusKeygenComplete
	statusFetching
	statusReady
	statusLinking
	statusBrowsingKeys
	statusSettingUsername
	statusQuitting
	statusError
)

// String prints the status as a string. This is just for debugging purposes.
func (s status) String() string {
	return [...]string{
		"initializing",
		"generating keys",
		"key generation complete",
		"fetching",
		"ready",
		"linking",
		"browsing keys",
		"setting username",
		"quitting",
		"error",
	}[s]
}

// menuChoice represents a chosen menu item
type menuChoice int

const (
	linkChoice menuChoice = iota
	keysChoice
	setUsernameChoice
	exitChoice
	unsetChoice // set when no choice has been made
)

// menu text corresponding to menu choices. these are presented to the user
var menuChoices = map[menuChoice]string{
	linkChoice:        "Link a machine",
	keysChoice:        "Manage linked keys",
	setUsernameChoice: "Set Username",
	exitChoice:        "Exit",
}

// MSG

type sshAuthErrorMsg struct{}

type sshAuthFailedMsg error

type newCharmClientMsg *charm.Client

type errMsg error

// MODEL

// Model holds the state for this program
type Model struct {
	cfg        *charm.Config
	cc         *charm.Client
	user       *charm.User
	err        error
	status     status
	menuIndex  int
	menuChoice menuChoice

	spinner  spinner.Model
	keygen   keygen.Model
	info     info.Model
	link     linkgen.Model
	username username.Model
	keys     keys.Model
}

// INIT

func initialize(cfg *charm.Config) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		s := spinner.NewModel()
		s.Type = spinner.Dot
		s.ForegroundColor = "244"
		m := Model{
			cfg:        cfg,
			cc:         nil,
			user:       nil,
			err:        nil,
			status:     statusInit,
			menuIndex:  0,
			menuChoice: unsetChoice,
			spinner:    s,
			keygen:     keygen.NewModel(),
		}
		return m, newCharmClient(m)
	}
}

// UPDATE

func update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m, ok := model.(Model)
	if !ok {
		return Model{
			err: errors.New("could not perform assertion on model in update"),
		}, nil
	}

	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	if m.cfg.Debug {
		if _, ok := msg.(spinner.TickMsg); !ok {
			log.Printf("STATUS: %s | MSG: %#v\n", m.status, msg)
		}
	}

	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.Type {
		case tea.KeyCtrlC:
			m.status = statusQuitting
			return m, tea.Quit
		}

		if m.status == statusReady { // Process keys for the menu

			switch msg.String() {

			// Quit
			case "q":
				fallthrough
			case "esc":
				m.status = statusQuitting
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

	case errMsg:
		m.status = statusError
		m.err = msg

	case spinner.TickMsg:
		m.spinner, _ = spinner.Update(msg, m.spinner)

	case sshAuthErrorMsg:
		m.status = statusKeygen
		return m, keygen.GenerateKeys

	case sshAuthFailedMsg:
		// TODO: report permanent failure
		return m, tea.Quit

	case keygen.DoneMsg:
		m.status = statusKeygenComplete
		return m, newCharmClient(m)

	case newCharmClientMsg:
		// Save reference to Charm client
		m.cc = msg

		// Initialize models that require a Charm client
		m.info = info.NewModel(m.cc)
		m.link = linkgen.NewModel(m.cc)
		m.username = username.NewModel(m.cc)
		m.keys = keys.NewModel(m.cc)

		// Fetch user info
		m.status = statusFetching
		return m, info.GetBio(m.cc)

	case info.GotBioMsg:
		m.status = statusReady
		m.info, _ = info.Update(msg, m.info)
		m.user = m.info.User

	case username.NameSetMsg:
		m.status = statusReady
		m.username = username.NewModel(m.cc) // reset the state
		m.info.User.Name = string(msg)

	}

	m, cmd = updateChilden(msg, m)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func updateChilden(msg tea.Msg, m Model) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.status {
	case statusKeygen:
		keygenModel, newCmd := keygen.Update(msg, tea.Model(m.keygen))
		mdl, ok := keygenModel.(keygen.Model)
		if !ok {
			m.err = errors.New("could not perform model assertion on keygen model")
			return m, nil
		}
		cmd = newCmd
		m.keygen = mdl
	case statusFetching:
		m.info, _ = info.Update(msg, m.info)
		if m.info.Quit {
			m.status = statusQuitting
			m.err = m.info.Err
			return m, tea.Quit
		}
		return m, nil
	case statusLinking:
		linkModel, _ := linkgen.Update(msg, tea.Model(m.link))
		mdl, ok := linkModel.(linkgen.Model)
		if !ok {
			m.err = errors.New("could not perform model assertion on link model")
			return m, nil
		}
		m.link = mdl
		if m.link.Exit {
			m.link = linkgen.NewModel(m.cc) // reset the state
			m.status = statusReady
		} else if m.link.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}
	case statusBrowsingKeys:
		var newModel tea.Model
		newModel, cmd = keys.Update(msg, m.keys)
		newKeysModel, ok := newModel.(keys.Model)
		if !ok {
			m.err = errors.New("could not perform model assertion on keys model")
			return m, nil
		}
		m.keys = newKeysModel
		if m.keys.Exit {
			m.keys = keys.NewModel(m.cc)
			m.status = statusReady
		} else if m.keys.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}
	case statusSettingUsername:
		m.username, cmd = username.Update(msg, m.username)
		if m.username.Done {
			m.username = username.NewModel(m.cc) // reset the state
			m.status = statusReady
		} else if m.username.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}
	}

	switch m.menuChoice {
	case linkChoice:
		m.status = statusLinking
		m.menuChoice = unsetChoice
		cmd = tea.Batch(linkgen.HandleLinkRequest(m.link)...)
	case keysChoice:
		m.status = statusBrowsingKeys
		m.menuChoice = unsetChoice
		cmd = keys.LoadKeys(m.cc)
	case setUsernameChoice:
		m.status = statusSettingUsername
		m.menuChoice = unsetChoice
	case exitChoice:
		m.status = statusQuitting
		cmd = tea.Quit
	}

	return m, cmd
}

// VIEW

func view(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		m.err = errors.New("could not perform assertion on model in view")
		m.status = statusError
	}

	s := charmLogoView()

	switch m.status {
	case statusInit:
		s += spinner.View(m.spinner) + " Initializing..."
	case statusKeygen:
		s += keygen.View(m.keygen)
	case statusKeygenComplete:
		s += spinner.View(m.spinner) + " Reinitializing..."
	case statusFetching:
		s += info.View(m.info)
	case statusReady:
		s += info.View(m.info)
		s += "\n\n" + menuView(m.menuIndex)
		s += footerView(m)
	case statusLinking:
		s += linkgen.View(m.link)
	case statusBrowsingKeys:
		s += keys.View(m.keys)
	case statusSettingUsername:
		s += username.View(m.username)
	case statusQuitting:
		s += quitView(m)
	case statusError:
		s += m.err.Error()
	}

	return indent.String(s, padding)
}

func charmLogoView() string {
	title := te.String(" Charm ").Foreground(common.Cream).Background(common.Color("#5A56E0")).String()
	return "\n" + title + "\n\n"
}

func menuView(currentIndex int) string {
	var s string
	for i := 0; i < len(menuChoices); i++ {
		e := "  "
		if i == currentIndex {
			e = te.String("> ").Foreground(common.Fuschia).String()
			e += te.String(menuChoices[menuChoice(i)]).Foreground(common.Fuschia).String()
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
	if m.err != nil {
		return errorView(m.err)
	}
	return common.HelpView("j/k, ↑/↓: choose", "enter: select")
}

func errorView(err error) string {
	head := te.String("Error: ").Foreground(common.Red).String()
	body := common.Subtle(err.Error())
	msg := common.Wrap(head + body)
	return "\n\n" + indent.String(msg, 2)
}

// COMMANDS

func newCharmClient(m Model) tea.Cmd {
	return func() tea.Msg {
		cc, err := charm.NewClient(m.cfg)
		if err == charm.ErrMissingSSHAuth {
			if m.status != statusKeygenComplete {
				return sshAuthErrorMsg{}
			}
			return sshAuthFailedMsg(err)
		} else if err != nil {
			// TODO: make this fatal
			return errMsg(err)
		}

		return newCharmClientMsg(cc)
	}
}

// SUBSCRIPTIONS

func subscriptions(model tea.Model) tea.Subs {
	m, ok := model.(Model)
	if !ok {
		// TODO: how can we handle this more gracefully?
		return nil
	}

	subs := tea.Subs{}

	switch m.status {
	case statusInit:
		subs["init-spinner-tick"] = tea.SubMap(spinner.Sub, m.spinner)
	case statusKeygen:
		s := keygen.Subscriptions(m.keygen)
		for k, v := range s {
			subs[k] = v
		}
	case statusFetching:
		subs["info-spinner-tick"] = info.Tick(m.info)
	case statusBrowsingKeys:
		subs["keys-spinner-tick"] = keys.Spin(m.keys)
	case statusSettingUsername:
		subs["username-input-blink"] = username.Blink(m.username)
		subs["username-spinner-tick"] = username.Spin(m.username)
	case statusLinking:
		subs["link-setup-spinner-tick"] = linkgen.Spin(m.link)
	}

	return subs
}
