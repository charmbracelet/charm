package keys

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui/common"
)

// wrap fingerprint to support additional states.
type fingerprint struct {
	client.Fingerprint
}

func (f fingerprint) state(s keyState, styles common.Styles) string {
	if s == keyDeleting {
		return fmt.Sprintf(
			"%s %s",
			styles.DeleteDim.Render(strings.ToUpper(f.Algorithm)),
			styles.Delete.Render(f.Type+":"+f.Value),
		)
	}
	return f.String()
}

type styledKey struct {
	styles      common.Styles
	date        string
	fingerprint fingerprint
	gutter      string
	keyLabel    string
	dateLabel   string
	dateVal     string
	note        string
}

func (m Model) newStyledKey(styles common.Styles, key charm.PublicKey, active bool) styledKey {
	date := key.CreatedAt.Format("02 Jan 2006 15:04:05 MST")
	fp, err := client.FingerprintSHA256(key)
	if err != nil {
		fp = client.Fingerprint{Value: "[error generating fingerprint]"}
	}

	var note string
	if active {
		note = m.styles.NoteDim.Render("• ") + m.styles.Note.Render("Current Key")
	}

	// Default state
	return styledKey{
		styles:      styles,
		date:        date,
		fingerprint: fingerprint{fp},
		gutter:      " ",
		keyLabel:    "Key:",
		dateLabel:   "Added:",
		dateVal:     styles.LabelDim.Render(date),
		note:        note,
	}
}

// Selected state
func (k *styledKey) selected() {
	k.gutter = common.VerticalLine(common.StateSelected)
	k.keyLabel = k.styles.Label.Render("Key:")
	k.dateLabel = k.styles.Label.Render("Added:")
}

// Deleting state
func (k *styledKey) deleting() {
	k.gutter = common.VerticalLine(common.StateDeleting)
	k.keyLabel = k.styles.Delete.Render("Key:")
	k.dateLabel = k.styles.Delete.Render("Added:")
	k.dateVal = k.styles.DeleteDim.Render(k.date)
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
		k.gutter, k.keyLabel, k.fingerprint.state(state, k.styles),
		k.gutter, k.dateLabel, k.dateVal, k.note,
	)
}
