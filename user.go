package charm

import (
	"crypto/sha1"
	"fmt"
	"time"
)

type User struct {
	ID        int        `json:"id"`
	CharmID   string     `json:"charm_id"`
	PublicKey *PublicKey `json:"public_key,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Bio       string     `json:"bio"`
	CreatedAt *time.Time `json:"created_at"`
}

type PublicKey struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id,omitempty"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
}

func (pk *PublicKey) Sha() string {
	return PublicKeySha(pk.Key)
}

func PublicKeySha(key string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(key)))
}
