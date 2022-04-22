package client

import (
	"crypto/sha256"
	"encoding/base64"
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
		Algorithm: algo(key.Type()),
		Type:      "SHA256",
		Value:     strings.TrimPrefix(ssh.FingerprintSHA256(key), "SHA256:"),
	}, nil
}

// RandomArt returns the randomart for the given key.
func RandomArt(k charm.PublicKey) (string, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(k.Key))
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	keyParts := strings.Split(string(ssh.MarshalAuthorizedKey(key)), " ")
	if len(keyParts) != 2 {
		return "", charm.ErrMalformedKey
	}

	b, err := base64.StdEncoding.DecodeString(keyParts[1])
	if err != nil {
		return "", err
	}

	h := sha256.New()
	_, _ = h.Write(b)
	board := randomart.GenerateSubtitled(
		h.Sum(nil),
		fmt.Sprintf(
			"%s %d",
			strings.ToUpper(algo(key.Type())),
			bitsize(key.Type()),
		),
		"SHA256",
	).String()
	return strings.TrimSpace(board), nil
}

func algo(keyType string) string {
	if idx := strings.Index(keyType, "@"); idx > 0 {
		return algo(keyType[0:idx])
	}
	parts := strings.Split(keyType, "-")
	if len(parts) == 2 {
		return parts[1]
	}
	if parts[0] == "sk" {
		return algo(strings.TrimPrefix(keyType, "sk-"))
	}
	return parts[0]
}

func bitsize(keyType string) int {
	switch keyType {
	case ssh.KeyAlgoED25519, ssh.KeyAlgoECDSA256, ssh.KeyAlgoSKECDSA256, ssh.KeyAlgoSKED25519:
		return 256
	case ssh.KeyAlgoECDSA384:
		return 384
	case ssh.KeyAlgoECDSA521:
		return 521
	case ssh.KeyAlgoDSA:
		return 1024
	case ssh.KeyAlgoRSA:
		return 3071 // usually
	default:
		return 0
	}
}
