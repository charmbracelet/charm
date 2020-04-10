package common

import (
	"strings"

	te "github.com/muesli/termenv"
)

// KeyValueView renders key-value pairs
func KeyValueView(stuff ...string) string {
	return SelectableKeyValueView(false, stuff...)
}

// SelectableKeyValueView renders key-value pairs as well as a parameter to
// mark this view as selected
func SelectableKeyValueView(selected bool, stuff ...string) string {
	if len(stuff) == 0 {
		return ""
	}
	var (
		s         string
		index     = 0
		pipeColor = "241"
	)
	if selected {
		pipeColor = YellowGreen
	}
	for i := 0; i < len(stuff); i++ {
		if i%2 == 0 {
			// even
			s += te.String("│ ").Foreground(Color(pipeColor)).String()
			if selected {
				s += stuff[i] + ": "
			} else {
				s += stuff[i] + ": "
			}
			continue
		}
		// odd
		s += te.String(stuff[i]).Foreground(Color(PurpleFg)).String()
		s += "\n"
		index++
	}
	return strings.TrimSpace(s)
}
