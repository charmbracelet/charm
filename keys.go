package charm

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/calmh/randomart"
	"github.com/charmbracelet/charm/ui/common"
)

// ErrMalformedKey parsing error for bad ssh key.
var ErrMalformedKey = errors.New("malformed key; is it missing the algorithm type at the beginning?")

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
		common.SubtleIndigoFg(strings.ToUpper(f.Algorithm)),
		common.IndigoFg(f.Type+":"+f.Value),
	)
}

// Key contains data and metadata for an SSH key.
type Key struct {
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// FingerprintSHA256 returns the algorithm and SHA256 fingerprint for the given
// key.
func (k Key) FingerprintSHA256() (Fingerprint, error) {
	keyParts := strings.Split(k.Key, " ")
	if len(keyParts) != 2 {
		return Fingerprint{}, ErrMalformedKey
	}

	b, err := base64.StdEncoding.DecodeString(keyParts[1])
	if err != nil {
		return Fingerprint{}, err
	}

	algo := strings.Replace(keyParts[0], "ssh-", "", -1)
	sha256sum := sha256.Sum256(b)
	hash := base64.RawStdEncoding.EncodeToString(sha256sum[:])

	return Fingerprint{
		Algorithm: algo,
		Type:      "SHA256",
		Value:     hash,
	}, nil
}

// RandomArt returns the randomart for the given key.
func (k Key) RandomArt() (string, error) {
	keyParts := strings.Split(k.Key, " ")
	if len(keyParts) != 2 {
		return "", ErrMalformedKey
	}

	b, err := base64.StdEncoding.DecodeString(keyParts[1])
	if err != nil {
		return "", err
	}

	algo := strings.ToUpper(strings.Replace(keyParts[0], "ssh-", "", -1))

	// TODO: also add bit size of key
	h := sha256.New()
	_, _ = h.Write(b)
	board := randomart.GenerateSubtitled(h.Sum(nil), algo, "SHA256").String()
	return strings.TrimSpace(board), nil
}
