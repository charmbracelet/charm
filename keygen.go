package charm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"
)

var (
	ErrMissingKeys = errors.New("missing one or more keys; did you forget to generate them?")
)

// SSHKeyPair holds a pair of SSH keys and associated methods
type SSHKeyPair struct {
	PrivateKeyPEM      []byte
	PublicKey          []byte
	bitSize            int
	keyDir             string
	privateKeyFilename string
	publicKeyFilename  string
}

// NewSSHKeyPair generates an SSHKeyPair, which contains a pair of SSH keys
func NewSSHKeyPair() (*SSHKeyPair, error) {
	s := &SSHKeyPair{
		bitSize:            4096,
		keyDir:             "~/.ssh",
		privateKeyFilename: "id_rsa",
		publicKeyFilename:  "id_rsa.pub",
	}
	err := s.GenerateKeys()
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GenerateKeys creates a public and private key pair
func (s *SSHKeyPair) GenerateKeys() error {
	var err error

	privateKey, err := generatePrivateKey(s.bitSize)
	if err != nil {
		return err
	}

	s.PublicKey, err = generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	s.PrivateKeyPEM = encodePrivateKeyToPEM(privateKey)
	return nil
}

// WriteKeys writes the SSH key pair to disk
func (s *SSHKeyPair) WriteKeys() error {
	if len(s.PrivateKeyPEM) == 0 || len(s.PublicKey) == 0 {
		return ErrMissingKeys
	}

	// Create directory if it doesn't exist
	if _, err := os.Stat(s.keyDir); err != nil && os.IsExist(err) {
		return err
	} else if err = os.Mkdir(s.keyDir, 0700); err != nil {
		return err
	}

	// Write keys to disk
	privPath := fmt.Sprintf("%s/%s", s.keyDir, s.privateKeyFilename)
	if err := writeKeyToFile(s.PrivateKeyPEM, privPath); err != nil {
		return err
	}
	pubPath := fmt.Sprintf("%s/%s", s.keyDir, s.publicKeyFilename)
	if err := writeKeyToFile(s.PublicKey, pubPath); err != nil {
		return err
	}

	return nil
}

// generatePrivateKey creates an RSA private key of a specified byte size, i.e.
// 2048, 4096, etc.
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate private key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// encodePrivateKeyToPEM encodes a private key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDir := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDir,
	}

	// Private key in PEM format
	privatePEMBytes := pem.EncodeToMemory(&privBlock)

	return privatePEMBytes
}

// generatePublicKey takes an RSA public key and returns bytes suitable for a
// public key file in the format "ssh-rsa ..."
func generatePublicKey(privateKey *rsa.PublicKey) ([]byte, error) {
	publicRSAKey, err := ssh.NewPublicKey(privateKey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRSAKey)
	return pubKeyBytes, nil
}

// writeKeyToFile write a key to a given path with appropriate permissions
func writeKeyToFile(keyBytes []byte, path string) error {
	if err := ioutil.WriteFile(path, keyBytes, 0600); err != nil {
		return err
	}
	return nil
}
