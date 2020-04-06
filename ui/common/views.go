package common

import (
	"strings"

	te "github.com/muesli/termenv"
)

// KeyValueView renders key-value pairs
func KeyValueView(m map[string]string) string {
	var s string
	for k, v := range m {
		s += te.String("│ ").Foreground(Color("241")).String()
		s += k + ": "
		s += te.String(v).Foreground(Color(purpleFg)).String()
		s += "\n"
	}
	return strings.TrimSpace(s)
}
