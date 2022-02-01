package keygen

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/charmbracelet/keygen"
	"github.com/muesli/reflow/indent"
)

const indentAmount = 2

// Status represents a keygen state
type Status int

// General states
const (
	StatusRunning Status = iota
	StatusError
	StatusSuccess
	StatusDone
	StatusQuitting
)

// NewProgram creates a new keygen TUI program
func NewProgram(host string, fancy bool) *tea.Program {
	m := NewModel()
	m.standalone = true
	m.fancy = fancy
	m.spinner = spinner.NewModel()
	m.spinner = common.NewSpinner()
	return tea.NewProgram(m)
}

// FailedMsg is a Bubble Tea message for keygen failure
type FailedMsg struct{ err error }

// Error returns the underlying error
func (f FailedMsg) Error() string { return f.err.Error() }

// SuccessMsg is a Bubble Tea message for keygen success
type SuccessMsg struct{}

// DoneMsg is sent when the keygen has completely finished running.
type DoneMsg struct{}

// Model is the Bubble Tea model which stores the state of the keygen.
type Model struct {
	Status        Status
	host          string
	styles        common.Styles
	err           error
	standalone    bool
	fancy         bool
	spinner       spinner.Model
	terminalWidth int
}

// NewModel returns a new keygen model in its initial state.
func NewModel() Model {
	return Model{
		Status: StatusRunning,
		styles: common.DefaultStyles(),
	}
}

// Init is the Bubble Tea initialization function for the keygen.
func (m Model) Init() tea.Cmd {
	return tea.Batch(GenerateKeys(m.host), spinner.Tick)
}

// Update is the Bubble Tea update loop for the keygen.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case FailedMsg:
		m.err = msg.err
		m.Status = StatusError
		return m, tea.Quit
	case SuccessMsg:
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
			newSpinnerModel, cmd := m.spinner.Update(msg)
			m.spinner = newSpinnerModel
			return m, cmd
		}
	}

	return m, nil
}

// View renders the view from the keygen model.
func (m Model) View() string {
	var s string

	switch m.Status {
	case StatusRunning:
		if m.standalone {
			s += m.spinner.View()
		}
		s += " Generating keys..."
	case StatusSuccess:
		s += m.styles.Checkmark.String() + "  Generated keys"
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

	if m.standalone && m.fancy {
		return indent.String(fmt.Sprintf("\n%s\n\n", s), indentAmount)
	}

	return s
}

// GenerateKeys is a Bubble Tea command that generates a pair of SSH keys and
// writes them to disk.
func GenerateKeys(host string) tea.Cmd {
	return func() tea.Msg {
		dp, err := client.DataPath(host)
		if err != nil {
			return FailedMsg{err}
		}
		_, err = keygen.NewWithWrite(dp, "charm", nil, keygen.Ed25519)
		if err != nil {
			return FailedMsg{err}
		}
		return SuccessMsg{}
	}
}

// pause runs the final pause before we wrap things up.
func pause() tea.Msg {
	time.Sleep(time.Millisecond * 600)
	return DoneMsg{}
}
