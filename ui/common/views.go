package common

import (
	"fmt"
	"strings"

	te "github.com/muesli/termenv"
)

func VerticalLine() string {
	return te.String("│").
		Foreground(ColorPair("#646464", "#BCBCBC")).
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
			s += fmt.Sprintf("%s %s: ", VerticalLine(), stuff[i])
			continue
		}
		// odd
		s += te.String(stuff[i]).Foreground(Indigo).String()
		s += "\n"
		index++
	}
	return strings.TrimSpace(s)
}
