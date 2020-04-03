package link

import (
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/tea"
	"github.com/muesli/reflow/wordwrap"
)

type Model struct {
	Quit bool // indicates the user wants to exit the whole program
	Exit bool // indicates the user wants to exit this mini-app
	cc   *charm.Client
}

// Reset resets the model to its initial state, except for the fact that it
// retains its reference to the Charm client
func (m *Model) Reset() {
	newModel := NewModel(m.cc)
	m = &newModel
}

// NewModel returns a new model
func NewModel(cc *charm.Client) Model {
	return Model{
		Quit: false,
		Exit: false,
		cc:   cc,
	}
}

// Update is the Tea update loop
func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Quit = true
			return m, nil
		case "q":
			fallthrough
		case "esc":
			m.Exit = true
			return m, nil
		}
	}

	return m, nil
}

func View(model Model) string {
	return wordwrap.String("You can link the SSH keys on another machine to your Charm account so both machines have access to your stuff. Rest assured that you can also unlink keys at any time.\n\nReady to go?", 50)
}
