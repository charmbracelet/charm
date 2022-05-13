package jwt

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"

	"gopkg.in/square/go-jose.v2"
)

// JWTKeyPair is an interface for JWT signing and verification.
type JWTKeyPair interface {
	PrivateKey() *ed25519.PrivateKey
	JWK() *jose.JSONWebKey
}

// JSONWebKeyPair holds the ED25519 private key and JSON Web Key used in JWT
// operations.
type JSONWebKeyPair struct {
	privateKey *ed25519.PrivateKey
	jwk        *jose.JSONWebKey
}

// PrivateKey implements the JWTKeyPair interface.
func (j JSONWebKeyPair) PrivateKey() *ed25519.PrivateKey {
	return j.privateKey
}

// JWK implements the JWTKeyPair interface.
func (j JSONWebKeyPair) JWK() *jose.JSONWebKey {
	return j.jwk
}

// NewJSONWebKeyPair creates a new JSONWebKeyPair from a given ED25519 private
// key.
func NewJSONWebKeyPair(pk *ed25519.PrivateKey) JWTKeyPair {
	sum := sha256.Sum256([]byte(*pk))
	kid := fmt.Sprintf("%x", sum)
	jwk := &jose.JSONWebKey{
		Key:       pk.Public(),
		KeyID:     kid,
		Algorithm: "EdDSA",
	}
	return JSONWebKeyPair{
		privateKey: pk,
		jwk:        jwk,
	}
}
