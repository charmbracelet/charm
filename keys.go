package charm

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/calmh/randomart"
)

// Key contains data and metadata for an SSH key
type Key struct {
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// Return the SHA256 fingerprint for the given key
func (k Key) FingerprintSHA256() (string, error) {
	b, err := base64.StdEncoding.DecodeString(k.Key)
	if err != nil {
		return "", err
	}
	sha256sum := sha256.Sum256(b)
	hash := base64.RawStdEncoding.EncodeToString(sha256sum[:])
	return fmt.Sprintf("SHA256:%s", hash), nil
}

// RandomArt returns the randomart for the given key
func (k Key) RandomArt() (string, error) {
	b, err := base64.StdEncoding.DecodeString(k.Key)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	_, _ = h.Write(b)
	board := randomart.GenerateSubtitled(h.Sum(nil), "", "SHA256").String()
	return strings.TrimSpace(board), nil
}
