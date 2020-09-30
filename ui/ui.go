package ui

import (
	"errors"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/charmclient"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/info"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/charm/ui/keys"
	"github.com/charmbracelet/charm/ui/linkgen"
	"github.com/charmbracelet/charm/ui/username"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const indentAmount = 2

// NewProgram returns a new Bubble Tea program. Use this to start up the
// Charm TUI.
func NewProgram(cfg *charm.Config) *tea.Program {
	if cfg.Logfile != "" {
		log.Println("-- Starting Charm ----------------")
		log.Println("Bubble Tea now initializing...")
	}
	return tea.NewProgram(initialize(cfg), update, view)
}

// status is used to indicate a high level application state.
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
	statusShowBackupInfo
	statusQuitting
	statusError
)

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
		"showing backup info",
		"quitting",
		"error",
	}[s]
}

// menuChoice represents a chosen menu item.
type menuChoice int

// menu choices
const (
	linkChoice menuChoice = iota
	keysChoice
	setUsernameChoice
	backupChoice
	exitChoice
	unsetChoice // set when no choice has been made
)

// menu text corresponding to menu choices. these are presented to the user.
var menuChoices = map[menuChoice]string{
	linkChoice:        "Link a machine",
	keysChoice:        "Manage linked keys",
	setUsernameChoice: "Set Username",
	backupChoice:      "Backup",
	exitChoice:        "Exit",
}

// Model holds the state for this program.
type model struct {
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

func initialize(cfg *charm.Config) func() (tea.Model, tea.Cmd) {
	return func() (tea.Model, tea.Cmd) {
		s := spinner.NewModel()
		s.Frames = spinner.Dot
		s.ForegroundColor = "244"

		m := model{
			cfg:        cfg,
			status:     statusInit,
			menuChoice: unsetChoice,
			spinner:    s,
			keygen:     keygen.NewModel(),
		}

		return m, tea.Batch(
			charmclient.NewClient(m.cfg),
			spinner.Tick(m.spinner),
		)
	}
}

func update(msg tea.Msg, mdl tea.Model) (tea.Model, tea.Cmd) {
	m, ok := mdl.(model)
	if !ok {
		return model{
			err: errors.New("could not perform assertion on model in update"),
		}, nil
	}

	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	if m.cfg.Debug && m.cfg.Logfile != "" {
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
			case "q", "esc":
				m.status = statusQuitting
				return m, tea.Quit

			// Prev menu item
			case "up", "k":
				m.menuIndex--
				if m.menuIndex < 0 {
					m.menuIndex = len(menuChoices) - 1
				}

			// Select menu item
			case "enter":
				m.menuChoice = menuChoice(m.menuIndex)

			// Next menu item
			case "down", "j":
				m.menuIndex++
				if m.menuIndex >= len(menuChoices) {
					m.menuIndex = 0
				}
			}
		}

	case spinner.TickMsg:
		switch m.status {
		case statusInit, statusKeygen, statusKeygenComplete, statusFetching:
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			return m, cmd
		}

	case charmclient.ErrMsg:
		m.status = statusError
		m.err = msg.Err

	case charmclient.SSHAuthErrorMsg:
		if m.status == statusInit {
			// SSH auth didn't work so let's try generating keys
			m.status = statusKeygen
			return m, keygen.GenerateKeys
		}
		// We tried the keygen, to no avail. Quit.
		m.err = msg.Err
		return m, tea.Quit

	case keygen.DoneMsg:
		m.status = statusKeygenComplete
		return m, tea.Batch(
			charmclient.NewClient(m.cfg),
			spinner.Tick(m.spinner),
		)

	case charmclient.NewClientMsg:
		// Save reference to Charm client
		m.cc = msg

		// Initialize models that require a Charm client
		m.info = info.NewModel(m.cc)
		m.username = username.NewModel(m.cc)
		m.keys = keys.NewModel(m.cfg)
		m.keys.SetCharmClient(m.cc)

		// Fetch user info
		m.status = statusFetching
		return m, info.GetBio(m.cc)

	case info.GotBioMsg:
		m.status = statusReady
		m.user = m.info.User
		m.info, cmd = info.Update(msg, m.info)
		cmds = append(cmds, cmd)

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

func updateChilden(msg tea.Msg, m model) (model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.status {
	// Keygen
	case statusKeygen:
		keygenModel, newCmd := keygen.Update(msg, tea.Model(m.keygen))
		mdl, ok := keygenModel.(keygen.Model)
		if !ok {
			m.err = errors.New("could not perform model assertion on keygen model")
			return m, nil
		}
		cmd = newCmd
		m.keygen = mdl

	// User info
	case statusFetching:
		m.info, cmd = info.Update(msg, m.info)
		if m.info.Quit {
			m.status = statusQuitting
			m.err = m.info.Err
			return m, tea.Quit
		}
		return m, cmd

	// Link generator
	case statusLinking:
		linkModel, cmd := linkgen.Update(msg, tea.Model(m.link))
		mdl, ok := linkModel.(linkgen.Model)
		if !ok {
			m.err = errors.New("could not perform model assertion on link model")
			return m, cmd
		}
		m.link = mdl
		if m.link.Exit {
			m.link = linkgen.NewModel() // reset the state
			m.status = statusReady
		} else if m.link.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}

	// Key browser
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
			m.keys = keys.NewModel(m.cfg)
			m.keys.SetCharmClient(m.cc)
			m.status = statusReady
		} else if m.keys.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}

	// Username tool
	case statusSettingUsername:
		m.username, cmd = username.Update(msg, m.username)
		if m.username.Done {
			m.username = username.NewModel(m.cc) // reset the state
			m.status = statusReady
		} else if m.username.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}

	case statusShowBackupInfo:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q":
				m.status = statusQuitting
				return m, tea.Quit
			case "esc":
				m.status = statusReady
				return m, nil
			}
		}
	}

	// Handle the menu
	switch m.menuChoice {
	case linkChoice:
		m.status = statusLinking
		m.menuChoice = unsetChoice
		m.link = linkgen.NewModel()
		m.link.SetCharmClient(m.cc)
		cmd = linkgen.InitLinkGen(m.link)

	case keysChoice:
		m.status = statusBrowsingKeys
		m.menuChoice = unsetChoice
		cmd = keys.LoadKeys(m.keys)

	case setUsernameChoice:
		m.status = statusSettingUsername
		m.menuChoice = unsetChoice
		cmd = username.InitialCmd(m.username)

	case backupChoice:
		m.status = statusShowBackupInfo
		m.menuChoice = unsetChoice

	case exitChoice:
		m.status = statusQuitting
		cmd = tea.Quit
	}

	return m, cmd
}

func view(mdl tea.Model) string {
	m, ok := mdl.(model)
	if !ok {
		m.err = errors.New("could not perform assertion on model in view")
		m.status = statusError
	}

	s := charmLogoView()

	switch m.status {
	case statusInit:
		s += spinner.View(m.spinner) + " Initializing..."
	case statusKeygen:
		if m.keygen.Status == keygen.StatusRunning {
			s += spinner.View(m.spinner)
		}
		s += keygen.View(m.keygen)
	case statusKeygenComplete:
		s += spinner.View(m.spinner) + " Reinitializing..."
	case statusFetching:
		if m.info.User == nil {
			s += spinner.View(m.spinner)
		}
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
	case statusShowBackupInfo:
		s += backupView(m)
	case statusQuitting:
		s += quitView(m)
	case statusError:
		s += m.err.Error()
	}

	return indent.String(s, indentAmount) + "\n"
}

func charmLogoView() string {
	title := te.String(" Charm ").Foreground(common.Cream.Color()).Background(common.Color("#5A56E0")).String()
	return "\n" + title + "\n\n"
}

func menuView(currentIndex int) string {
	var s string
	for i := 0; i < len(menuChoices); i++ {
		e := "  "
		if i == currentIndex {
			e = te.String("> ").Foreground(common.Fuschia.Color()).String()
			e += te.String(menuChoices[menuChoice(i)]).Foreground(common.Fuschia.Color()).String()
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

func backupView(m model) string {
	p, err := charm.DataPath()
	if err != nil {
		return errorView(err)
	}
	s := "Your Charm account uses SSH keys specific to Charm. These keys are automatically cut the first time you authenticate. It’s " + te.String("very important").Bold().String() + " that you keep these keys safe as they’re the keys to your account.\n\n"
	s += "You can make a quick backup of your keys by running:\n\n"
	s += "  " + common.Code("charm backup-keys") + "\n\n"
	s += "Your keys can also be found at:\n\n"
	s += "  " + common.Keyword(p) + "\n\n"
	s += "For more info see " + common.Code("charm backup-keys -h") + ". We’ll be adding more recovery features in the future.\n\n"
	s += common.HelpView("esc: back", "q: quit") + "\n\n"
	return common.Wrap(s)
}

func quitView(m model) string {
	if m.err != nil {
		return fmt.Sprintf("Uh oh, there’s been an error: %v\n", m.err)
	}
	return "Thanks for using Charm!\n"
}

func footerView(m model) string {
	if m.err != nil {
		return errorView(m.err)
	}
	return "\n\n" + common.HelpView("j/k, ↑/↓: choose", "enter: select")
}

func errorView(err error) string {
	head := te.String("Error: ").Foreground(common.Red.Color()).String()
	body := common.Subtle(err.Error())
	msg := common.Wrap(head + body)
	return "\n\n" + indent.String(msg, 2)
}
