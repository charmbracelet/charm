package main

import "github.com/charmbracelet/tea"

type MenuModel struct {
	Choice MenuChoice // user's chosen menu item
	Index  int        // cursor index
}

// Choice represents a menu item choice
type MenuChoice int

// Choices
const (
	Link MenuChoice = iota
	Keys
	CopyID
	JWT
	Username
)

var menuChoices = map[MenuChoice]string{
	Link:     "Link this Computer",
	Keys:     "List Keys",
	CopyID:   "Copy Charm ID",
	JWT:      "Get Token",
	Username: "Change Username",
}

type Menu struct{}

func (menu Menu) Update(msg tea.Msg, m MenuModel) (MenuModel, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch msg.String() {

		// Prev menu item
		case "up":
			fallthrough
		case "k":
			m.Index--
			if m.Index < 0 {
				m.Index = len(menuChoices) - 1
			}
			return m, nil

		// Next menu item
		case "down":
			fallthrough
		case "j":
			m.Index++
			if m.Index >= len(menuChoices) {
				m.Index = 0
			}
			return m, nil

		// Choose menu item
		case "enter":
			m.Choice = MenuChoice(m.Index)
			return m, nil

		default:
			return m, nil
		}

	default:
		return m, nil
	}

}

// View renders the menu
func (menu *Menu) View(m MenuModel) string {

	s := "\n\nWhat do you want to do?\n\n"

	for i := 0; i < len(menuChoices); i++ {
		e := "  "
		if i == m.Index {
			e = "> "
		}
		e += menuChoices[MenuChoice(i)] + "\n"
		s += e
	}

	return s
}
