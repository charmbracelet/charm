package ui

import (
	"errors"
	"fmt"
	"log"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/charm/ui/info"
	"github.com/charmbracelet/charm/ui/keygen"
	"github.com/charmbracelet/charm/ui/keys"
	"github.com/charmbracelet/charm/ui/linkgen"
	"github.com/charmbracelet/charm/ui/username"
	"github.com/muesli/reflow/indent"
	te "github.com/muesli/termenv"
)

const padding = 2

// NewProgram returns a new boba program
func NewProgram(cfg *charm.Config) *boba.Program {
	return boba.NewProgram(initialize(cfg), update, view)
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

func initialize(cfg *charm.Config) func() (boba.Model, boba.Cmd) {
	return func() (boba.Model, boba.Cmd) {
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
		return m, boba.Batch(
			newCharmClient(m),
			spinner.Tick(m.spinner),
		)
	}
}

// UPDATE

func update(msg boba.Msg, model boba.Model) (boba.Model, boba.Cmd) {
	m, ok := model.(Model)
	if !ok {
		return Model{
			err: errors.New("could not perform assertion on model in update"),
		}, nil
	}

	var (
		cmds []boba.Cmd
		cmd  boba.Cmd
	)

	if m.cfg.Debug {
		if _, ok := msg.(spinner.TickMsg); !ok {
			log.Printf("STATUS: %s | MSG: %#v\n", m.status, msg)
		}
	}

	switch msg := msg.(type) {

	case boba.KeyMsg:

		switch msg.Type {
		case boba.KeyCtrlC:
			m.status = statusQuitting
			return m, boba.Quit
		}

		if m.status == statusReady { // Process keys for the menu

			switch msg.String() {

			// Quit
			case "q":
				fallthrough
			case "esc":
				m.status = statusQuitting
				return m, boba.Quit

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
		if m.status == statusInit || m.status == statusKeygenComplete {
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			return m, cmd
		}

	case sshAuthErrorMsg:
		m.status = statusKeygen
		return m, keygen.InitialCmd(m.keygen)

	case sshAuthFailedMsg:
		// TODO: report permanent failure
		return m, boba.Quit

	case keygen.DoneMsg:
		m.status = statusKeygenComplete
		return m, boba.Batch(
			newCharmClient(m),
			spinner.Tick(m.spinner),
		)

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

	return m, boba.Batch(cmds...)
}

func updateChilden(msg boba.Msg, m Model) (Model, boba.Cmd) {
	var cmd boba.Cmd

	switch m.status {
	case statusKeygen:
		keygenModel, newCmd := keygen.Update(msg, boba.Model(m.keygen))
		mdl, ok := keygenModel.(keygen.Model)
		if !ok {
			m.err = errors.New("could not perform model assertion on keygen model")
			return m, nil
		}
		cmd = newCmd
		m.keygen = mdl
	case statusFetching:
		m.info, cmd = info.Update(msg, m.info)
		if m.info.Quit {
			m.status = statusQuitting
			m.err = m.info.Err
			return m, boba.Quit
		}
		return m, cmd
	case statusLinking:
		linkModel, cmd := linkgen.Update(msg, boba.Model(m.link))
		mdl, ok := linkModel.(linkgen.Model)
		if !ok {
			m.err = errors.New("could not perform model assertion on link model")
			return m, cmd
		}
		m.link = mdl
		if m.link.Exit {
			m.link = linkgen.NewModel(m.cc) // reset the state
			m.status = statusReady
		} else if m.link.Quit {
			m.status = statusQuitting
			return m, boba.Quit
		}
	case statusBrowsingKeys:
		var newModel boba.Model
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
			return m, boba.Quit
		}
	case statusSettingUsername:
		m.username, cmd = username.Update(msg, m.username)
		if m.username.Done {
			m.username = username.NewModel(m.cc) // reset the state
			m.status = statusReady
		} else if m.username.Quit {
			m.status = statusQuitting
			return m, boba.Quit
		}
	}

	switch m.menuChoice {
	case linkChoice:
		m.status = statusLinking
		m.menuChoice = unsetChoice
		cmd = boba.Batch(linkgen.HandleLinkRequest(m.link)...)
		cmd = linkgen.InitialCmd(m.link)
	case keysChoice:
		m.status = statusBrowsingKeys
		m.menuChoice = unsetChoice
		cmd = keys.InitialCmd(m.keys)
	case setUsernameChoice:
		m.status = statusSettingUsername
		m.menuChoice = unsetChoice
		cmd = username.InitialCmd(m.username)
	case exitChoice:
		m.status = statusQuitting
		cmd = boba.Quit
	}

	return m, cmd
}

// VIEW

func view(model boba.Model) string {
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

	return indent.String(s, padding) + "\n"
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
	head := te.String("Error: ").Foreground(common.Red.Color()).String()
	body := common.Subtle(err.Error())
	msg := common.Wrap(head + body)
	return "\n\n" + indent.String(msg, 2)
}

// COMMANDS

func newCharmClient(m Model) boba.Cmd {
	return func() boba.Msg {
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
