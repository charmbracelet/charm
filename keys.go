package charm

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/calmh/randomart"
)

var ErrMalformedKey = errors.New("malformed key; is it missing the algorithm type at the beginning?")

// Key contains data and metadata for an SSH key
type Key struct {
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Return the SHA256 fingerprint for the given key
func (k Key) FingerprintSHA256() (string, error) {
	keyParts := strings.Split(k.Key, " ")
	if len(keyParts) != 2 {
		return "", ErrMalformedKey
	}

	b, err := base64.StdEncoding.DecodeString(keyParts[1])
	if err != nil {
		return "", err
	}
	sha256sum := sha256.Sum256(b)
	hash := base64.RawStdEncoding.EncodeToString(sha256sum[:])
	return fmt.Sprintf("SHA256:%s", hash), nil
}

// RandomArt returns the randomart for the given key
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
