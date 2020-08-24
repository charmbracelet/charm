package keygen

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/keygen"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
)

const indentAmount = 2

type status int

// General states
const (
	StatusRunning status = iota
	StatusError
	StatusSuccess
	StatusDone
	StatusQuitting
)

type failedMsg struct{ err error }
type successMsg struct{}

// DoneMsg is sent when the keygen has completely finished running.
type DoneMsg struct{}

// Model is the Bubble Tea model which stores the state of the keygen.
type Model struct {
	Status        status
	err           error
	standalone    bool
	spinner       spinner.Model
	terminalWidth int
}

// Init is the Bubble Tea initialization function for the keygen.
func Init() (tea.Model, tea.Cmd) {
	m := NewModel()
	m.standalone = true

	m.spinner = spinner.NewModel()
	m.spinner.Frames = spinner.Dot
	m.spinner.ForegroundColor = common.SpinnerColor.String()

	return m, tea.Batch(GenerateKeys, spinner.Tick(m.spinner))
}

// NewModel returns a new keygen model in its initial state.
func NewModel() Model {
	return Model{
		Status: StatusRunning,
	}
}

// Update is the Bubble Tea update loop for the keygen.
func Update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m, ok := model.(Model)
	if !ok {
		return model, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.Status = StatusQuitting
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		return m, nil
	case failedMsg:
		m.err = msg.err
		m.Status = StatusError
		return m, tea.Quit
	case successMsg:
		m.Status = StatusSuccess
		return m, pause
	case DoneMsg:
		if m.standalone {
			return m, tea.Quit
		}
		m.Status = StatusDone
		return m, nil
	case spinner.TickMsg:
		if m.Status == StatusRunning {
			newSpinnerModel, cmd := spinner.Update(msg, m.spinner)
			m.spinner = newSpinnerModel
			return m, cmd
		}
	}

	return m, nil
}

// View renders the view from the keygen model.
func View(model tea.Model) string {
	m, ok := model.(Model)
	if !ok {
		return "could not perform assertion on model in view"
	}

	var s string

	switch m.Status {
	case StatusRunning:
		if m.standalone {
			s += spinner.View(m.spinner)
		}
		s += " Generating keys..."
	case StatusSuccess:
		s += termenv.String("✔").Foreground(common.Green.Color()).String()
		s += "  Generated keys"
	case StatusError:
		switch m.err.(type) {
		case keygen.SSHKeysAlreadyExistErr:
			s += "You already have SSH keys :)"
		default:
			s += fmt.Sprintf("Uh oh, there's been an error: %v", m.err)
		}
	case StatusQuitting:
		s += "Exiting..."
	}

	if m.standalone {
		return wordwrap.String(
			indent.String(fmt.Sprintf("\n%s\n\n", s), indentAmount),
			m.terminalWidth-(indentAmount*2),
		)
	}

	return s
}

// GenerateKeys is a Bubble Tea command that generates a pair of SSH keys and
// writes them to disk.
func GenerateKeys() tea.Msg {
	_, err := keygen.NewSSHKeyPair(nil)
	if err != nil {
		return failedMsg{err}
	}
	return successMsg{}
}

// pause runs the final pause before we wrap things up.
func pause() tea.Msg {
	time.Sleep(time.Millisecond * 600)
	return DoneMsg{}
}
