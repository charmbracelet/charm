package common

import (
	"fmt"
	"strings"

	te "github.com/muesli/termenv"
)

// State is a general UI state used to help style components
type State int

// possible states
const (
	StateNormal State = iota
	StateSelected
	StateDeleting
)

// VerticalLine return a vertical line colored according to the given state
func VerticalLine(state State) string {
	var c te.Color
	switch state {
	case StateSelected:
		c = ColorPair("#F684FF", "#F684FF")
	case StateDeleting:
		c = ColorPair("#893D4E", "#FF8BA7")
	default:
		c = ColorPair("#646464", "#BCBCBC")
	}
	return te.String("│").
		Foreground(c).
		String()
}

// KeyValueView renders key-value pairs
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
		s += te.String(stuff[i]).Foreground(Indigo).String()
		s += "\n"
		index++
	}
	return strings.TrimSpace(s)
}
