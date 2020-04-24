package common

import (
	"github.com/muesli/reflow/wordwrap"
	te "github.com/muesli/termenv"
)

const (
	PurpleBg    = "#5A56E0"
	PurpleFg    = "#7571F9"
	YellowGreen = "#ECFD65"

	wrapAt = 60
)

var (
	// Color wraps termenv.ColorProfile.Color, which produces a termenv color
	// for use in termenv styling.
	Color func(string) te.Color = te.ColorProfile().Color

	HasDarkBackground = te.HasDarkBackground()
	SpinnerColor      string
	Indigo            = ColorPair("#7571F9", "#5A56E0")
	Cream             = ColorPair("#FFFDF5", "#FFFDF5")
	Fuschia           = ColorPair("#EE6FF8", "#EE6FF8")
	Green             = ColorPair("#04B575", "#04B575")
	Red               = ColorPair("#ED567A", "#FF4672")
	FaintRed          = ColorPair("#C74665", "#FF6F91")
)

func init() {
	if HasDarkBackground {
		SpinnerColor = "#747373"
	} else {
		SpinnerColor = "#8E8E8E"
	}
}

func ColorPair(dark, light string) te.Color {
	if HasDarkBackground {
		return Color(dark)
	}
	return Color(light)
}

// Wrap wraps lines at a predefined width via package muesli/reflow.
func Wrap(s string) string {
	return wordwrap.String(s, wrapAt)
}

// Keyword applies special formatting to imporant words or phrases
func Keyword(s string) string {
	return te.String(s).Foreground(ColorPair("#ECFD65", "#04B575")).String()
}

// Code applies special formatting to strings indeded to read as code
func Code(s string) string {
	return te.String(" " + s + " ").
		Foreground(ColorPair("#ED567A", "#FF4672")).
		Background(ColorPair("#2B2A2A", "#EBE5EC")).
		String()
}

// Subtle applies formatting to strings intended to be "subtle"
func Subtle(s string) string {
	return te.String(s).Foreground(ColorPair("#5C5C5C", "#9B9B9B")).String()
}

// HELP

// Help renders text intended to display at help text, usually at the bottom of
// a view.
func HelpView(sections ...string) string {
	var s = "\n\n"
	if len(sections) == 0 {
		return s
	}
	for i := 0; i < len(sections); i++ {
		s += te.String(sections[i]).Foreground(ColorPair("#5C5C5C", "#9B9B9B")).String()
		if i < len(sections)-1 {
			s += helpDivider()
		}
	}
	return s
}

func helpDivider() string {
	return te.String(" • ").Foreground(ColorPair("#3C3C3C", "#DDDADA")).String()
}

// BUTTONS

// Button view renders something that resembles a button
func ButtonView(text string, focused bool) string {
	return buttonStyling(text, false, focused)
}

func YesButtonView(focused bool) string {
	return buttonStyling("  ", false, focused) +
		buttonStyling("Y", true, focused) +
		buttonStyling("es  ", false, focused)
}

func NoButtonView(focused bool) string {
	return buttonStyling("  ", false, focused) +
		buttonStyling("N", true, focused) +
		buttonStyling("o  ", false, focused)
}

func OKButtonView(focused bool, defaultButton bool) string {
	return buttonStyling("  ", false, focused) +
		buttonStyling("OK", defaultButton, focused) +
		buttonStyling("  ", false, focused)
}

func CancelButtonView(focused bool, defaultButton bool) string {
	return buttonStyling("  ", false, focused) +
		buttonStyling("Cancel", defaultButton, focused) +
		buttonStyling("  ", false, focused)
}

func buttonStyling(str string, underline, focused bool) string {
	var s te.Style = te.String(str).Foreground(Cream)
	if focused {
		s = s.Background(Fuschia)
	} else {
		s = s.Background(ColorPair("#827983", "#BDB0BE"))
	}
	if underline {
		s = s.Underline()
	}
	return s.String()
}
