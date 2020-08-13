package common

import (
	"github.com/muesli/reflow/wordwrap"
	te "github.com/muesli/termenv"
)

const (
	wrapAt = 60
)

var (
	// Color wraps termenv.ColorProfile.Color, which produces a termenv color
	// for use in termenv styling.
	Color func(string) te.Color = te.ColorProfile().Color
)

var (
	// HasDarkBackground stores whether or not the terminal has a dark
	// background.
	HasDarkBackground = te.HasDarkBackground()

	// SpinnerColor stores the color of the spinner.
	SpinnerColor string
)

// Colors for dark and light backgrounds.
var (
	Indigo       ColorPair = NewColorPair("#7571F9", "#5A56E0")
	SubtleIndigo           = NewColorPair("#514DC1", "#7D79F6")
	Cream                  = NewColorPair("#FFFDF5", "#FFFDF5")
	YellowGreen            = NewColorPair("#ECFD65", "#04B575")
	Fuschia                = NewColorPair("#EE6FF8", "#EE6FF8")
	Green                  = NewColorPair("#04B575", "#04B575")
	Red                    = NewColorPair("#ED567A", "#FF4672")
	FaintRed               = NewColorPair("#C74665", "#FF6F91")
	NoColor                = NewColorPair("", "")
)

// Functions for styling strings.
var (
	IndigoFg       func(string) string = te.Style{}.Foreground(Indigo.Color()).Styled
	SubtleIndigoFg                     = te.Style{}.Foreground(SubtleIndigo.Color()).Styled
	RedFg                              = te.Style{}.Foreground(Red.Color()).Styled
	FaintRedFg                         = te.Style{}.Foreground(FaintRed.Color()).Styled
)

func init() {
	if HasDarkBackground {
		SpinnerColor = "#747373"
	} else {
		SpinnerColor = "#8E8E8E"
	}
}

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

// Keyword applies special formatting to imporant words or phrases
func Keyword(s string) string {
	return te.String(s).Foreground(Green.Color()).String()
}

// Code applies special formatting to strings indeded to read as code
func Code(s string) string {
	return te.String(" " + s + " ").
		Foreground(NewColorPair("#ED567A", "#FF4672").Color()).
		Background(NewColorPair("#2B2A2A", "#EBE5EC").Color()).
		String()
}

// Subtle applies formatting to strings intended to be "subtle"
func Subtle(s string) string {
	return te.String(s).Foreground(NewColorPair("#5C5C5C", "#9B9B9B").Color()).String()
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

// Button view renders something that resembles a button.
func ButtonView(text string, focused bool) string {
	return buttonStyling(text, false, focused)
}

// YesButtonView return a button reading "Yes".
func YesButtonView(focused bool) string {
	return buttonStyling("  ", false, focused) +
		buttonStyling("Y", true, focused) +
		buttonStyling("es  ", false, focused)
}

// NoButtonView returns a button reading "No."
func NoButtonView(focused bool) string {
	return buttonStyling("  ", false, focused) +
		buttonStyling("N", true, focused) +
		buttonStyling("o  ", false, focused)
}

// OKButtonView returns a button reading "OK".
func OKButtonView(focused bool, defaultButton bool) string {
	return buttonStyling("  ", false, focused) +
		buttonStyling("OK", defaultButton, focused) +
		buttonStyling("  ", false, focused)
}

// CancelButtonView returns a button reading "Cancel."
func CancelButtonView(focused bool, defaultButton bool) string {
	return buttonStyling("  ", false, focused) +
		buttonStyling("Cancel", defaultButton, focused) +
		buttonStyling("  ", false, focused)
}

func buttonStyling(str string, underline, focused bool) string {
	var s te.Style = te.String(str).Foreground(Cream.Color())
	if focused {
		s = s.Background(Fuschia.Color())
	} else {
		s = s.Background(ColorPair{"#827983", "#BDB0BE"}.Color())
	}
	if underline {
		s = s.Underline()
	}
	return s.String()
}
