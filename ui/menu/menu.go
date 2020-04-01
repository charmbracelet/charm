package menu

import (
	"github.com/charmbracelet/tea"
)

// Choice represents a menu item choice
type Choice int

// Choices
const (
	CopyCharmID Choice = iota
	SetUsername
	Link
	Keys
	JWT

	Unset = -1
)

var choices = map[Choice]string{
	//Link:        "Link this Computer",
	//Keys:        "List Keys",
	CopyCharmID: "Copy Charm ID",
	//JWT:         "Get Token",
	SetUsername: "Change Username",
}

type Model struct {
	Choice Choice // user's chosen menu item
	Index  int    // cursor index
}

func NewModel() Model {
	return Model{Choice: Unset}
}

func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.String() {

		// Prev menu item
		case "up":
			fallthrough
		case "k":
			m.Index--
			if m.Index < 0 {
				m.Index = len(choices) - 1
			}
			return m, nil

		// Next menu item
		case "down":
			fallthrough
		case "j":
			m.Index++
			if m.Index >= len(choices) {
				m.Index = 0
			}
			return m, nil

		// Choose menu item
		case "enter":
			m.Choice = Choice(m.Index)
			return m, nil

		default:
			return m, nil
		}

	default:
		return m, nil
	}

}

// View renders the menu
func View(m Model) string {

	s := "What do you want to do?\n\n"

	for i := 0; i < len(choices); i++ {
		e := "  "
		if i == m.Index {
			e = "> "
		}
		e += choices[Choice(i)]
		if i < len(choices)-1 {
			e += "\n"
		}
		s += e
	}

	return s
}
