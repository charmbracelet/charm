package menu

import "github.com/charmbracelet/tea"

type Model struct {
	Choice Choice // user's chosen menu item
	Index  int    // cursor index
}

// Choice represents a menu item choice
type Choice int

// Choices
const (
	Link Choice = iota
	Keys
	CopyID
	JWT
	Username
)

var choices = map[Choice]string{
	Link:     "Link this Computer",
	Keys:     "List Keys",
	CopyID:   "Copy Charm ID",
	JWT:      "Get Token",
	Username: "Change Username",
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

	s := "\n\nWhat do you want to do?\n\n"

	for i := 0; i < len(choices); i++ {
		e := "  "
		if i == m.Index {
			e = "> "
		}
		e += choices[Choice(i)] + "\n"
		s += e
	}

	return s
}
