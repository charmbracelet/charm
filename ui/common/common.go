package common

import (
	te "github.com/muesli/termenv"
)

var (
	// Color wraps termenv.ColorProfile.Color, which produces a termenv color
	// for use in termenv styling.
	Color func(string) te.Color = te.ColorProfile().Color

	purpleBg = "#5A56E0"
	purpleFg = "#7571F9"
)
