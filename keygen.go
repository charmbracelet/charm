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
	// MissingSSHKeysErr indicates we're missing
	MissingSSHKeysErr = errors.New("missing one or more keys; did you forget to generate them?")
)

// SSHKeyGenFilesystemError
type SSHKeygenFilesystemError struct {
	error
}

// Error implements the error interface
func (e SSHKeygenFilesystemError) Error() string {
	return e.error.Error()
}

// SSHKeyPair holds a pair of SSH keys and associated methods
type SSHKeyPair struct {
	PrivateKeyPEM []byte
	PublicKey     []byte
	bitSize       int
	keyDir        string
	filename      string // private key filename; public key will have .pub appended
}

// NewSSHKeyPair generates an SSHKeyPair, which contains a pair of SSH keys
func NewSSHKeyPair() (*SSHKeyPair, error) {
	return newSSHKeyPairWithBitSize(4096)
}

// newSSHKeyPairWithBitSize returns an SSH key pair with a given bit size. This
// is implemented for quick testing only. In production, use NewSSHKeyPair.
func newSSHKeyPairWithBitSize(bitSize int) (*SSHKeyPair, error) {
	s := &SSHKeyPair{
		bitSize:  bitSize,
		keyDir:   "~/.ssh",
		filename: "id_rsa",
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
		return MissingSSHKeysErr
	}

	// Create directory if it doesn't exist + make sure permissions are right
	if err := createSSHDirectory(s.keyDir); err != nil {
		return err
	}

	// Write keys to disk
	privPath := fmt.Sprintf("%s/%s", s.keyDir, s.filename)
	if err := writeKeyToFile(s.PrivateKeyPEM, privPath); err != nil {
		return err
	}
	pubPath := fmt.Sprintf("%s/%s", s.keyDir, s.filename+".pub")
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
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return ioutil.WriteFile(path, keyBytes, 0600)
	}
	return SSHKeygenFilesystemError{fmt.Errorf("file %s already exists", keyBytes)}
}

// createSSHDirectory creates a directory if it doesn't exist, and makes
// sure the permissions are correct for SSH keys if it does
func createSSHDirectory(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Create directory
		return os.Mkdir(path, 0700)
	}

	if err != nil {
		// Some other error
		return err
	}

	if !info.IsDir() {
		// It's not a directory
		return SSHKeygenFilesystemError{fmt.Errorf("%s is not a directory", path)}
	}

	if info.Mode().Perm() != 0700 {
		// Fix permissions
		if err := os.Chmod(path, 0700); err != nil {
			return err
		}
	}

	return nil
}
