package keys

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/ui/common"
	te "github.com/muesli/termenv"
)

const (
	lineChar = "│"
)

var (
	yellowGreen = common.Color(common.YellowGreen)
	purpleFg    = common.Color(common.PurpleFg)
	hotPink     = common.Color("204")
	dullHotPink = common.Color("168")
	gray        = common.Color("241")
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
	fp, err := sha256Fingerprint(key.Key)
	if err != nil {
		fp = "[error generating fingerprint]"
	}

	var note string
	if active {
		bullet := te.String("• ").Foreground(common.Color("241")).String()
		note = bullet + te.String("Current Key").Foreground(common.Color("35")).String()
	}

	// Default state
	return styledKey{
		date:        date,
		fingerprint: fp,
		line:        te.String(lineChar).Foreground(gray).String(),
		keyLabel:    "Key:",
		keyVal:      te.String(fp).Foreground(purpleFg).String(),
		dateLabel:   "Added:",
		dateVal:     te.String(date).Foreground(purpleFg).String(),
		note:        note,
	}
}

// Selected state
func (k *styledKey) selected() {
	k.line = te.String(lineChar).Foreground(yellowGreen).String()
}

// Deleting state
func (k *styledKey) deleting() {
	k.line = te.String(lineChar).Foreground(yellowGreen).String()
	k.keyLabel = te.String("Key:").Foreground(hotPink).String()
	k.keyVal = te.String(k.fingerprint).Foreground(dullHotPink).String()
	k.dateLabel = te.String("Added:").Foreground(hotPink).String()
	k.dateVal = te.String(k.date).Foreground(dullHotPink).String()
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

func truncate(s string, n int) string {
	if len(s) > n {
		if n > 3 {
			n -= 3
		}
		return s[0:n] + "..."
	}
	return s
}

// sha256Fingerprint creates a SHA256 fingerprint from a given base 64 key
func sha256Fingerprint(pubKey string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(pubKey)
	sha256sum := sha256.Sum256(b)
	hash := base64.RawStdEncoding.EncodeToString(sha256sum[:])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SHA256:%s", hash), nil
}
