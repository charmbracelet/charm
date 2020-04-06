package common

import (
	"github.com/muesli/reflow/wordwrap"
	te "github.com/muesli/termenv"
)

var (
	// Color wraps termenv.ColorProfile.Color, which produces a termenv color
	// for use in termenv styling.
	Color func(string) te.Color = te.ColorProfile().Color

	cream             = "#FFFDF5"
	purpleBg          = "#5A56E0"
	purpleFg          = "#7571F9"
	fuschia           = "#EE6FF8"
	yellowGreen       = "#ECFD65"
	unfocusedButtonBg = "#827983"
	focusedButtonBg   = fuschia

	wrapAt = 60
)

// Wrap wraps lines at a predefined width via package muesli/reflow.
func Wrap(s string) string {
	return wordwrap.String(s, wrapAt)
}

// Keyword applies special formatting to imporant words or phrases
func Keyword(s string) string {
	return te.String(s).Foreground(Color(fuschia)).String()
}

// Code applies special formatting to strings indeded to read as code
func Code(s string) string {
	return te.String(" " + s + " ").Foreground(Color("203")).Background(Color("237")).String()
}

// Help renders text intended to display at help text, usually at the bottom of
// a view.
func HelpView(s string) string {
	return "\n\n" + te.String(s).Foreground(Color("241")).String()
}

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

func buttonStyling(str string, underline, focused bool) string {
	var s te.Style = te.String(str).Foreground(Color(cream))
	if focused {
		s = s.Background(Color(focusedButtonBg))
	} else {
		s = s.Background(Color(unfocusedButtonBg))
	}
	if underline {
		s = s.Underline()
	}
	return s.String()
}
