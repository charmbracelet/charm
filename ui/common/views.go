package common

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
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

var lineColors = map[State]lipgloss.TerminalColor{
	StateNormal:   lipgloss.AdaptiveColor{Light: "#BCBCBC", Dark: "#646464"},
	StateSelected: lipgloss.Color("#F684FF"),
	StateDeleting: lipgloss.AdaptiveColor{Light: "#FF8BA7", Dark: "#893D4E"},
	StateSpecial:  lipgloss.Color("#04B575"),
}

// VerticalLine return a vertical line colored according to the given state.
func VerticalLine(state State) string {
	return lipgloss.NewStyle().
		SetString("│").
		Foreground(lineColors[state]).
		String()
}

var valStyle = lipgloss.NewStyle().Foreground(indigo)

var (
	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#8E8E8E", Dark: "#747373"})

	blurredButtonStyle = lipgloss.NewStyle().
				Foreground(cream).
				Background(lipgloss.AdaptiveColor{Light: "#BDB0BE", Dark: "#827983"}).
				Padding(0, 3)

	focusedButtonStyle = blurredButtonStyle.Copy().
				Background(fuschia)
)

// NewSpinner returns a spinner model.
func NewSpinner() spinner.Model {
	s := spinner.NewModel()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return s
}

// KeyValueView renders key-value pairs.
func KeyValueView(stuff ...string) string {
	if len(stuff) == 0 {
		return ""
	}

	var (
		s     string
		index int
	)
	for i := 0; i < len(stuff); i++ {
		if i%2 == 0 {
			// even: key
			s += fmt.Sprintf("%s %s: ", VerticalLine(StateNormal), stuff[i])
			continue
		}
		// odd: value
		s += valStyle.Render(stuff[i])
		s += "\n"
		index++
	}

	return strings.TrimSpace(s)
}

var (
	helpDivider = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#3C3C3C"}).
			Padding(0, 1).
			Render("•")

	helpSection = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})
)

// HelpView renders text intended to display at help text, often at the
// bottom of a view.
func HelpView(sections ...string) string {
	var s string
	if len(sections) == 0 {
		return s
	}

	for i := 0; i < len(sections); i++ {
		s += helpSection.Render(sections[i])
		if i < len(sections)-1 {
			s += helpDivider
		}
	}

	return s
}

// ButtonView renders something that resembles a button.
func ButtonView(text string, focused bool) string {
	return styledButton(text, false, focused)
}

// YesButtonView return a button reading "Yes".
func YesButtonView(focused bool) string {
	var st lipgloss.Style
	if focused {
		st = focusedButtonStyle
	} else {
		st = blurredButtonStyle
	}
	return underlineInitialCharButton("Yes", st)
}

// NoButtonView returns a button reading "No.".
func NoButtonView(focused bool) string {
	var st lipgloss.Style
	if focused {
		st = focusedButtonStyle
	} else {
		st = blurredButtonStyle
	}
	st = st.Copy().
		PaddingLeft(st.GetPaddingLeft() + 1).
		PaddingRight(st.GetPaddingRight() + 1)
	return underlineInitialCharButton("No", st)
}

func underlineInitialCharButton(str string, style lipgloss.Style) string {
	if len(str) == 0 {
		return ""
	}

	var (
		r     = []rune(str)
		left  = r[0]
		right = r[1:]
	)

	leftStyle := style.Copy().Underline(true).UnsetPaddingRight()
	rightStyle := style.Copy().UnsetPaddingLeft()

	return leftStyle.Render(string(left)) + rightStyle.Render(string(right))
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
