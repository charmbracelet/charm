package common

import (
	"os"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	te "github.com/muesli/termenv"
)

const wrapAt = 60

var (
	// Color wraps termenv.ColorProfile.Color, which produces a termenv color
	// for use in termenv styling.
	Color func(string) te.Color = te.ColorProfile().Color

	// HasDarkBackground stores whether or not the terminal has a dark
	// background.
	HasDarkBackground = te.HasDarkBackground()
)

// Colors for dark and light backgrounds.
var (
	Indigo       ColorPair = NewColorPair("#7571F9", "#5A56E0")
	SubtleIndigo           = NewColorPair("#514DC1", "#7D79F6")
	Cream                  = lipgloss.AdaptiveColor{Light: "#FFFDF5", Dark: "#FFFDF5"}
	YellowGreen            = NewColorPair("#ECFD65", "#04B575")
	Fuschia                = lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	Green                  = NewColorPair("#04B575", "#04B575")
	Red                    = NewColorPair("#ED567A", "#FF4672")
	FaintRed               = NewColorPair("#C74665", "#FF6F91")
	NoColor                = NewColorPair("", "")
)

var (
	CursorStyle = lipgloss.NewStyle().
			Foreground(Fuschia)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#8E8E8E", Dark: "#747373"})

	blurredButtonStyle = lipgloss.NewStyle().
				Foreground(Cream).
				Background(lipgloss.AdaptiveColor{Light: "#BDB0BE", Dark: "#827983"}).
				Padding(0, 2)

	focusedButtonStyle = blurredButtonStyle.Copy().
				Background(Fuschia)
)

// NewSpinner returns a spinner model.
func NewSpinner() spinner.Model {
	s := spinner.NewModel()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return s
}

// Functions for styling strings.
var (
	IndigoFg       func(string) string = te.Style{}.Foreground(Indigo.Color()).Styled
	SubtleIndigoFg                     = te.Style{}.Foreground(SubtleIndigo.Color()).Styled
	RedFg                              = te.Style{}.Foreground(Red.Color()).Styled
	FaintRedFg                         = te.Style{}.Foreground(FaintRed.Color()).Styled
)

// ColorPair is a pair of colors, one intended for a dark background and the
// other intended for a light background. We'll automatically determine which
// of these colors to use.
type ColorPair struct {
	Dark  string
	Light string
}

// NewColorPair is a helper function for creating a ColorPair.
func NewColorPair(dark, light string) ColorPair {
	return ColorPair{dark, light}
}

// Color returns the appropriate termenv.Color for the terminal background.
func (c ColorPair) Color() te.Color {
	if HasDarkBackground {
		return Color(c.Dark)
	}

	return Color(c.Light)
}

// String returns a string representation of the color appropriate for the
// current terminal background.
func (c ColorPair) String() string {
	if HasDarkBackground {
		return c.Dark
	}

	return c.Light
}

// Wrap wraps lines at a predefined width via package muesli/reflow.
func Wrap(s string) string {
	return wordwrap.String(s, wrapAt)
}

// Keyword applies special formatting to imporant words or phrases.
func Keyword(s string) string {
	return te.String(s).Foreground(Green.Color()).String()
}

// Code applies special formatting to strings indeded to read as code.
func Code(s string) string {
	return te.String(" " + s + " ").
		Foreground(NewColorPair("#ED567A", "#FF4672").Color()).
		Background(NewColorPair("#2B2A2A", "#EBE5EC").Color()).
		String()
}

// Subtle applies formatting to strings intended to be "subtle".
func Subtle(s string) string {
	return te.String(s).Foreground(NewColorPair("#5C5C5C", "#9B9B9B").Color()).String()
}

// Format long descriptions with indentation
func FormatLong(s string) string {
	return indent.String(wordwrap.String("\n"+s, wrapAt-2), 2)
}

var (
	isTTY    bool
	checkTTY sync.Once
)

// Returns true if standard out is a terminal
func IsTTY() bool {
	checkTTY.Do(func() {
		isTTY = isatty.IsTerminal(os.Stdout.Fd())
	})
	return isTTY
}
