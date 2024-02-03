package server

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"

	jose "github.com/go-jose/go-jose"
)

// JSONWebKeyPair holds the ED25519 private key and JSON Web Key used in JWT
// operations.
type JSONWebKeyPair struct {
	PrivateKey *ed25519.PrivateKey
	JWK        jose.JSONWebKey
}

// NewJSONWebKeyPair creates a new JSONWebKeyPair from a given ED25519 private
// key.
func NewJSONWebKeyPair(pk *ed25519.PrivateKey) JSONWebKeyPair {
	sum := sha256.Sum256([]byte(*pk))
	kid := fmt.Sprintf("%x", sum)
	jwk := jose.JSONWebKey{
		Key:       pk.Public(),
		KeyID:     kid,
		Algorithm: "EdDSA",
	}
	return JSONWebKeyPair{
		PrivateKey: pk,
		JWK:        jwk,
	}
}
