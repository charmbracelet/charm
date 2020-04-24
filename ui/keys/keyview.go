package keys

import (
	"fmt"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	te "github.com/muesli/termenv"
)

type styledKey struct {
	date        string
	fingerprint string
	line        string
	keyLabel    string
	keyVal      string
	dateLabel   string
	dateVal     string
	note        string
}

func newStyledKey(key charm.Key, active bool) styledKey {
	date := key.CreatedAt.Format("02 Jan 2006 15:04:05 MST")
	fp, err := key.FingerprintSHA256()
	if err != nil {
		fp = "[error generating fingerprint]"
	}

	var note string
	if active {
		bullet := te.String("• ").Foreground(common.ColorPair("#2B4A3F", "#ABE5D1")).String()
		note = bullet + te.String("Current Key").Foreground(common.ColorPair("#04B575", "#04B575")).String()
	}

	// Default state
	return styledKey{
		date:        date,
		fingerprint: fp,
		line:        common.VerticalLine(common.StateNormal),
		keyLabel:    "Key:",
		keyVal:      te.String(fp).Foreground(common.Indigo).String(),
		dateLabel:   "Added:",
		dateVal:     te.String(date).Foreground(common.Indigo).String(),
		note:        note,
	}
}

// Selected state
func (k *styledKey) selected() {
	k.line = common.VerticalLine(common.StateSelected)
	k.keyLabel = te.String("Key:").Foreground(common.Fuschia).String()
	k.dateLabel = te.String("Added:").Foreground(common.Fuschia).String()
}

// Deleting state
func (k *styledKey) deleting() {
	k.line = common.VerticalLine(common.StateDeleting)
	k.keyLabel = te.String("Key:").Foreground(common.Red).String()
	k.keyVal = te.String(k.fingerprint).Foreground(common.FaintRed).String()
	k.dateLabel = te.String("Added:").Foreground(common.Red).String()
	k.dateVal = te.String(k.date).Foreground(common.FaintRed).String()
}

func (k styledKey) render(state keyState) string {
	switch state {
	case keySelected:
		k.selected()
	case keyDeleting:
		k.deleting()
	}
	return fmt.Sprintf(
		"%s %s %s\n%s %s %s %s\n\n",
		k.line, k.keyLabel, k.keyVal,
		k.line, k.dateLabel, k.dateVal, k.note,
	)
}
