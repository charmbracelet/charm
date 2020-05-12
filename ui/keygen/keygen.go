package keygen

import (
	"fmt"
	"time"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/spinner"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/termenv"
)

type status int

const (
	statusRunning status = iota
	statusError
	statusSuccess
	statusDone
	statusQuitting
)

// MSG

type failedMsg error

type successMsg struct{}

type DoneMsg struct{}

// MODEL

type Model struct {
	status     status
	err        error
	spinner    spinner.Model
	standalone bool
}

// INIT

func Init() (boba.Model, boba.Cmd) {
	m := NewModel()
	m.standalone = true
	return m, InitialCmd(m)
}

func NewModel() Model {
	s := spinner.NewModel()
	s.Type = spinner.Dot
	s.ForegroundColor = common.SpinnerColor
	return Model{
		status:     statusRunning,
		err:        nil,
		spinner:    s,
		standalone: false,
	}
}

func InitialCmd(m Model) boba.Cmd {
	return boba.Batch(GenerateKeys, spinner.Tick(m.spinner))
}

// UPDATE

func Update(msg boba.Msg, model boba.Model) (boba.Model, boba.Cmd) {
	m, ok := model.(Model)
	if !ok {
		return model, nil
	}

	var cmd boba.Cmd

	switch msg := msg.(type) {
	case boba.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			fallthrough
		case "esc":
			fallthrough
		case "q":
			m.status = statusQuitting
			return m, boba.Quit
		}
	case failedMsg:
		m.err = msg
		m.status = statusError
		return m, boba.Quit
	case successMsg:
		m.status = statusSuccess
		return m, pause
	case spinner.TickMsg:
		if m.status == statusRunning {
			m.spinner, cmd = spinner.Update(msg, m.spinner)
			return m, cmd
		}
	case DoneMsg:
		if m.standalone {
			return m, boba.Quit
		}
		m.status = statusDone
		return m, nil
	}

	return m, nil
}

// VIEWS

func View(model boba.Model) string {
	m, ok := model.(Model)
	if !ok {
		return "could not perform assertion on model in view"
	}

	var s string

	switch m.status {
	case statusRunning:
		s += fmt.Sprintf("%s Generating keys...", spinner.View(m.spinner))
	case statusSuccess:
		s += termenv.String("✔").Foreground(common.Green.Color()).String()
		s += "  Generated keys"
	case statusError:
		s += fmt.Sprintf("Uh oh, there's been an error: %v", m.err)
	case statusQuitting:
		s += "Exiting..."
	}

	if m.standalone {
		return indent.String(fmt.Sprintf("\n%s\n", s), 2)
	}

	return s
}

// COMMANDS

// GenerateKeys is a Boba command that generates a pair of SSH keys and writes
// them to disk
func GenerateKeys() boba.Msg {
	_, err := charm.NewSSHKeyPair()
	if err != nil {
		return failedMsg(err)
	}
	return successMsg{}
}

// pause runs the final pause before we wrap things up
func pause() boba.Msg {
	time.Sleep(time.Millisecond * 600)
	return DoneMsg{}
}
