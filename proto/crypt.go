package proto

import (
	"time"
)

// EncryptKey is the symmetric key used to encrypt data for a Charm user. An
// encrypt key will be encoded for every public key associated with a user's
// Charm account.
type EncryptKey struct {
	ID        string     `json:"id"`
	Key       string     `json:"key"`
	PublicKey string     `json:"public_key,omitempty"`
	CreatedAt *time.Time `json:"created_at"`
}
