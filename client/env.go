package client

import (
	"path/filepath"

	gap "github.com/muesli/go-app-paths"
)

// DataPath returns the Charm data path for the current user. This is where
// Charm keys are stored.
func DataPath(host string) (string, error) {
	scope := gap.NewScope(gap.User, filepath.Join("charm", host))
	dataPath, err := scope.DataPath("")
	if err != nil {
		return "", err
	}
	return dataPath, nil
}
