// Package ui implements Bubble Tea TUIs for Charm use-cases.
package ui

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui/charmclient"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/info"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/charm/ui/keys"
	"github.com/charmbracelet/charm/ui/linkgen"
	"github.com/charmbracelet/charm/ui/username"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
)

// NewProgram returns a new Bubble Tea program. Use this to start up the
// Charm TUI.
func NewProgram(cfg *client.Config) *tea.Program {
	if cfg.Logfile != "" {
		log.Println("-- Starting Charm ----------------")
		log.Println("Bubble Tea now initializing...")
	}
	return tea.NewProgram(initialModel(cfg))
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
	cfg        *client.Config
	cc         *client.Client
	styles     common.Styles
	user       *charm.User
	err        error
	status     status
	menuIndex  int
	menuChoice menuChoice

	spinner  spinner.Model
	keygen   keygen.Model
	info     info.Model
	linkgen  linkgen.Model
	username username.Model
	keys     keys.Model

	terminalWidth int
}

func initialModel(cfg *client.Config) model {
	return model{
		cfg:        cfg,
		styles:     common.DefaultStyles(),
		status:     statusInit,
		menuChoice: unsetChoice,
		spinner:    common.NewSpinner(),
		keygen:     keygen.NewModel(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		charmclient.NewClient(m.cfg),
		spinner.Tick,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width

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
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case charmclient.ErrMsg:
		m.status = statusError
		m.err = msg.Err

	case charmclient.SSHAuthErrorMsg:
		if m.status == statusInit {
			// SSH auth didn't work so let's try generating keys
			m.status = statusKeygen
			return m, keygen.GenerateKeys(m.cfg.Host)
		}
		// We tried the keygen, to no avail. Quit.
		m.err = msg.Err
		return m, tea.Quit

	case keygen.DoneMsg:
		m.status = statusKeygenComplete
		return m, tea.Batch(
			charmclient.NewClient(m.cfg),
			spinner.Tick,
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
		newModel, newCmd := m.keygen.Update(msg)
		keygenModel, ok := newModel.(keygen.Model)
		if !ok {
			panic("could not perform assertion on keygen model")
		}
		m.keygen = keygenModel
		cmd = newCmd

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
		newModel, newCmd := m.linkgen.Update(msg)
		linkgenModel, ok := newModel.(linkgen.Model)
		if !ok {
			panic("could not peform assertion on linkgen model")
		}
		m.linkgen = linkgenModel
		cmd = newCmd

		if m.linkgen.Exit {
			m.linkgen = linkgen.NewModel(m.cfg) // reset the state
			m.status = statusReady
		} else if m.linkgen.Quit {
			m.status = statusQuitting
			return m, tea.Quit
		}

	// Key browser
	case statusBrowsingKeys:
		newModel, newCmd := m.keys.Update(msg)
		keysModel, ok := newModel.(keys.Model)
		if !ok {
			panic("could not perform assertion on keys model")
		}
		m.keys = keysModel
		cmd = newCmd

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
		m.linkgen = linkgen.NewModel(m.cfg)
		m.linkgen.SetCharmClient(m.cc)
		cmd = linkgen.InitLinkGen(m.linkgen)

	case keysChoice:
		m.status = statusBrowsingKeys
		m.menuChoice = unsetChoice
		cmd = keys.LoadKeys(m.keys)

	case setUsernameChoice:
		m.status = statusSettingUsername
		m.menuChoice = unsetChoice
		cmd = username.InitialCmd()

	case backupChoice:
		m.status = statusShowBackupInfo
		m.menuChoice = unsetChoice

	case exitChoice:
		m.status = statusQuitting
		cmd = tea.Quit
	}

	return m, cmd
}

func (m model) View() string {
	w := m.terminalWidth - m.styles.App.GetHorizontalFrameSize()
	s := m.styles.Logo.String() + "\n\n"

	switch m.status {
	case statusInit:
		s += m.spinner.View() + " Initializing..."
	case statusKeygen:
		if m.keygen.Status == keygen.StatusRunning {
			s += m.spinner.View()
		}
		s += m.keygen.View()
	case statusKeygenComplete:
		s += m.spinner.View() + " Reinitializing..."
	case statusFetching:
		if m.info.User == nil {
			s += m.spinner.View()
		}
		s += m.info.View()
	case statusReady:
		s += m.info.View()
		s += "\n\n" + m.menuView()
		s += footerView(m)
	case statusLinking:
		s += m.linkgen.View()
	case statusBrowsingKeys:
		s += m.keys.View()
	case statusSettingUsername:
		s += username.View(m.username)
	case statusShowBackupInfo:
		s += m.backupView()
	case statusQuitting:
		s += m.quitView()
	case statusError:
		s += m.err.Error()
	}

	return m.styles.App.Render(wrap.String(wordwrap.String(s, w), w))
}

func (m model) menuView() string {
	var s string
	for i := 0; i < len(menuChoices); i++ {
		e := "  "
		menuItem := menuChoices[menuChoice(i)]
		if i == m.menuIndex {
			e = m.styles.SelectionMarker.String() +
				m.styles.SelectedMenuItem.Render(menuItem)
		} else {
			e += menuItem
		}
		if i < len(menuChoices)-1 {
			e += "\n"
		}
		s += e
	}

	return s
}

func (m model) backupView() string {
	keyword := m.styles.Keyword.Render
	code := m.styles.Code.Render
	em := lipgloss.NewStyle().Underline(true).Render

	p, err := client.DataPath(m.cfg.Host)
	if err != nil {
		return m.errorView(err)
	}
	s := "Your Charm account uses SSH keys specific to Charm. These keys are automatically cut the first time you authenticate. It’s " + em("very important") + " that you keep these keys safe as they’re the keys to your account.\n\n"
	s += "You can make a quick backup of your keys by running:\n\n"
	s += "  " + code("charm backup-keys") + "\n\n"
	s += "Your keys can also be found at:\n\n"
	s += "  " + keyword(p) + "\n\n"
	s += "For more info see " + code("charm backup-keys -h") + ". We’ll be adding more recovery features in the future.\n\n"
	s += common.HelpView("esc: back", "q: quit") + "\n\n"
	return m.styles.Wrap.Render(s)
}

func (m model) quitView() string {
	if m.err != nil {
		return fmt.Sprintf("Uh oh, there’s been an error: %s\n", m.err)
	}
	return "Thanks for using Charm!\n"
}

func footerView(m model) string {
	if m.err != nil {
		return m.errorView(m.err)
	}
	return "\n\n" + common.HelpView("j/k, ↑/↓: choose", "enter: select")
}

func (m model) errorView(err error) string {
	head := m.styles.Error.Render("Error: ")
	body := m.styles.Subtle.Render(err.Error())
	msg := m.styles.Wrap.Render(head + body)
	return "\n\n" + indent.String(msg, 2)
}
