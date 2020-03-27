package menu

import "github.com/charmbracelet/tea"

var (
	choices = []string{
		"Link this Computer",
		"List Keys",
		"Copy Charm ID",
		"Get Token",
		"Change Username",
	}
)

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

// ChoiceMsg contains a Choice corresponding to a menu item choice
type ChoiceMsg Choice

// Model is the model for this menu
type Model struct {
	Choice Choice
	index  int
}

// NewModel returns default model for the menu
func NewModel() Model {
	return Model{
		index: 0,
	}
}

// Update is the main Tea update loop for this menu
func Update(msg tea.Msg, m Model) (Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.String() {

		// Prev menu item
		case "up":
			fallthrough
		case "k":
			m.index--
			if m.index < 0 {
				m.index = len(choices) - 1
			}
			return m, nil

		// Next menu item
		case "down":
			fallthrough
		case "j":
			m.index++
			if m.index >= len(choices) {
				m.index = 0
			}
			return m, nil

		// Choose menu item
		case "enter":
			m.Choice = Choice(m.index)
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

	for i, v := range choices {
		e := "  "
		if i == m.index {
			e = "> "
		}
		e += v + "\n"
		s += e
	}

	return s
}
