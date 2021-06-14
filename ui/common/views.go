package common

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	te "github.com/muesli/termenv"
)

// State is a general UI state used to help style components.
type State int

// UI states.
const (
	StateNormal State = iota
	StateSelected
	StateActive
	StateSpecial
	StateDeleting
)

// VerticalLine return a vertical line colored according to the given state.
func VerticalLine(state State) string {
	var c te.Color
	switch state {
	case StateSelected:
		c = NewColorPair("#F684FF", "#F684FF").Color()
	case StateDeleting:
		c = NewColorPair("#893D4E", "#FF8BA7").Color()
	case StateActive:
		c = NewColorPair("#9BA92F", "#6CCCA9").Color()
	case StateSpecial:
		c = NewColorPair("#04B575", "#04B575").Color()
	default:
		c = NewColorPair("#646464", "#BCBCBC").Color()
	}

	return te.String("│").
		Foreground(c).
		String()
}

// KeyValueView renders key-value pairs.
func KeyValueView(stuff ...string) string {
	if len(stuff) == 0 {
		return ""
	}

	var (
		s     string
		index = 0
	)
	for i := 0; i < len(stuff); i++ {
		if i%2 == 0 {
			// even
			s += fmt.Sprintf("%s %s: ", VerticalLine(StateNormal), stuff[i])
			continue
		}
		// odd
		s += te.String(stuff[i]).Foreground(Indigo.Color()).String()
		s += "\n"
		index++
	}

	return strings.TrimSpace(s)
}

// HELP

// HelpView renders text intended to display at help text, often at the
// bottom of a view.
func HelpView(sections ...string) string {
	var s string
	if len(sections) == 0 {
		return s
	}

	for i := 0; i < len(sections); i++ {
		s += te.String(sections[i]).Foreground(NewColorPair("#5C5C5C", "#9B9B9B").Color()).String()
		if i < len(sections)-1 {
			s += helpDivider()
		}
	}

	return s
}

func helpDivider() string {
	return te.String(" • ").Foreground(NewColorPair("#3C3C3C", "#DDDADA").Color()).String()
}

// BUTTONS

// ButtonView renders something that resembles a button.
func ButtonView(text string, focused bool) string {
	return styledButton(text, false, focused)
}

// YesButtonView return a button reading "Yes".
func YesButtonView(focused bool) string {
	str := lipgloss.NewStyle().Underline(true).Render("Y") + "es"
	return styledButton(str, false, focused)
}

// NoButtonView returns a button reading "No.".
func NoButtonView(focused bool) string {
	str := lipgloss.NewStyle().Underline(true).Render("N") + "o"
	return styledButton(str, false, focused)
}

// OKButtonView returns a button reading "OK".
func OKButtonView(focused bool, defaultButton bool) string {
	return styledButton("OK", defaultButton, focused)
}

// CancelButtonView returns a button reading "Cancel.".
func CancelButtonView(focused bool, defaultButton bool) string {
	return styledButton("Cancel", defaultButton, focused)
}

func styledButton(str string, underlined, focused bool) string {
	var st lipgloss.Style
	if focused {
		st = focusedButtonStyle.Copy()
	} else {
		st = blurredButtonStyle.Copy()
	}
	if underlined {
		st = st.Underline(true)
	}
	return st.Render(str)
}
