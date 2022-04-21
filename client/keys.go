package client

import (
	"fmt"
	"strings"

	"github.com/calmh/randomart"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/charm/ui/common"
	"golang.org/x/crypto/ssh"
)

var styles = common.DefaultStyles()

// Fingerprint is the fingerprint of an SSH key.
type Fingerprint struct {
	Algorithm string
	Type      string
	Value     string
}

// String outputs a string representation of the fingerprint.
func (f Fingerprint) String() string {
	return fmt.Sprintf(
		"%s %s",
		styles.ListDim.Render(strings.ToUpper(f.Algorithm)),
		styles.ListKey.Render(f.Type+":"+f.Value),
	)
}

// FingerprintSHA256 returns the algorithm and SHA256 fingerprint for the given
// key.
func FingerprintSHA256(k charm.PublicKey) (Fingerprint, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(k.Key))
	if err != nil {
		return Fingerprint{}, fmt.Errorf("failed to parse public key: %w", err)
	}

	return Fingerprint{
		Algorithm: strings.TrimPrefix(key.Type(), "ssh-"),
		Type:      "SHA256",
		Value:     strings.TrimPrefix(ssh.FingerprintSHA256(key), "SHA256:"),
	}, nil
}

// RandomArt returns the randomart for the given key.
func RandomArt(k charm.PublicKey) (string, error) {
	finger, err := FingerprintSHA256(k)
	if err != nil {
		return "", err
	}

	// TODO: also add bit size of key
	board := randomart.GenerateSubtitled([]byte(finger.Value), finger.Algorithm, finger.Type).String()
	return strings.TrimSpace(board), nil
}
