package proto

import (
	"crypto/sha1" // nolint: gosec
	"fmt"
	"time"
)

// User represents a Charm user account.
type User struct {
	ID        int        `json:"id"`
	CharmID   string     `json:"charm_id"`
	PublicKey *PublicKey `json:"public_key,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Bio       string     `json:"bio"`
	CreatedAt *time.Time `json:"created_at"`
}

// PublicKey represents to public SSH key for a Charm user.
type PublicKey struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id,omitempty"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
}

// Sha returns the SHA for the public key in hex format.
func (pk *PublicKey) Sha() string {
	return PublicKeySha(pk.Key)
}

// PublicKeySha returns the SHA for a public key in hex format.
func PublicKeySha(key string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(key))) // nolint: gosec
}
