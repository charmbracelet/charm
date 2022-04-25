package common

import (
	"os"
	"sync"

	"github.com/mattn/go-isatty"
)

var (
	isTTY    bool
	checkTTY sync.Once
)

// Returns true if standard out is a terminal.
func IsTTY() bool {
	checkTTY.Do(func() {
		isTTY = isatty.IsTerminal(os.Stdout.Fd())
	})
	return isTTY
}
