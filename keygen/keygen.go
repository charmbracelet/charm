package keygen

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/charmbracelet/charm"
	"github.com/mikesmitty/edkey"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/crypto/ssh"
)

const rsaDefaultBits = 4096

// ErrMissingSSHKeys indicates we're missing some keys that we expected to
// have after generating. This should be an extreme edge case.
var ErrMissingSSHKeys = errors.New("missing one or more keys; did something happen to them after they were generated?")

// SSHKeysAlreadyExistErr indicates that files already exist at the location at
// which we're attempting to create SSH keys.
type SSHKeysAlreadyExistErr struct {
	path string
}

// Error returns the a human-readable error message for SSHKeysAlreadyExistErr.
// It satisfies the error interface.
func (e SSHKeysAlreadyExistErr) Error() string {
	return fmt.Sprintf("ssh key %s already exists", e.path)
}

// FilesystemError is used to signal there was a problem creating keys at the
// filesystem-level. For example, when we're unable to create a directory to
// store new SSH keys in.
type FilesystemErr struct {
	error
}

// Error returns a human-readable string for the erorr. It implements the error
// interface.
func (e FilesystemErr) Error() string {
	return e.error.Error()
}

// Unwrap returne the underlying error.
func (e FilesystemErr) Unwrap() error {
	return e.error
}

// SSHKeyPair holds a pair of SSH keys and associated methods.
type SSHKeyPair struct {
	PrivateKeyPEM []byte
	PublicKey     []byte
	KeyDir        string
	Filename      string // private key filename; public key will have .pub appended
}

func (s SSHKeyPair) privateKeyPath() string {
	return filepath.Join(s.KeyDir, s.Filename)
}

func (s SSHKeyPair) publicKeyPath() string {
	return filepath.Join(s.KeyDir, s.Filename+".pub")
}

// NewSSHKeyPair generates an SSHKeyPair, which contains a pair of SSH keys.
// The keys are written to disk.
func NewSSHKeyPair(passphrase []byte) (*SSHKeyPair, error) {
	s := &SSHKeyPair{}
	if err := s.GenerateRSAKeys(rsaDefaultBits, passphrase); err != nil {
		return nil, err
	}
	if err := s.WriteKeys(); err != nil {
		return nil, err
	}
	return s, nil
}

// GenerateEd25519Keys creates a pair of EdD25519 keys for SSH auth.
func (s *SSHKeyPair) GenerateEd25519Keys() error {
	// Generate keys
	pubKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	// Encode PEM
	pemBlock := pem.EncodeToMemory(&pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: edkey.MarshalED25519PrivateKey(privateKey),
	})

	// Prepare public key
	publicKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return err
	}

	dataPath, err := charm.DataPath()
	if err != nil {
		return err
	}

	// serialize for public key file on disk
	serializedPublicKey := ssh.MarshalAuthorizedKey(publicKey)

	s.PrivateKeyPEM = pemBlock
	s.PublicKey = pubKeyWithMemo(serializedPublicKey)
	s.KeyDir = dataPath
	s.Filename = "charm_ed25519"
	return nil
}

// GenerateRSAKeys creates a pair for RSA keys for SSH auth.
func (s *SSHKeyPair) GenerateRSAKeys(bitSize int, passphrase []byte) error {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return err
	}

	// Validate private key
	err = privateKey.Validate()
	if err != nil {
		return err
	}

	// Get ASN.1 DER format
	x509Encoded := x509.MarshalPKCS1PrivateKey(privateKey)

	block := &pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509Encoded,
	}

	// encrypt private key with passphrase
	if len(passphrase) > 0 {
		block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, passphrase, x509.PEMCipherAES256)
		if err != nil {
			return err
		}
	}

	// Private key in PEM format
	pemBlock := pem.EncodeToMemory(block)

	// Generate public key
	publicRSAKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return err
	}

	dataPath, err := charm.DataPath()
	if err != nil {
		return err
	}

	// serialize for public key file on disk
	serializedPubKey := ssh.MarshalAuthorizedKey(publicRSAKey)

	s.PrivateKeyPEM = pemBlock
	s.PublicKey = pubKeyWithMemo(serializedPubKey)
	s.KeyDir = dataPath
	s.Filename = "charm_rsa"
	return nil
}

// PrepFilesystem makes sure the state of the filesystem is as it needs to be
// in order to write our keys to disk. It will create and/or set permissions on
// the SSH directory we're going to write our keys to (for example, ~/.ssh) as
// well as make sure that no files exist at the location in which we're going
// to write out keys.
func (s *SSHKeyPair) PrepFilesystem() error {
	var err error

	s.KeyDir, err = homedir.Expand(s.KeyDir)
	if err != nil {
		return err
	}

	info, err := os.Stat(s.KeyDir)
	if os.IsNotExist(err) {
		// Directory doesn't exist: create it
		return os.MkdirAll(s.KeyDir, 0700)
	}
	if err != nil {
		// There was another error statting the directory; something is awry
		return FilesystemErr{err}
	}
	if !info.IsDir() {
		// It exists but it's not a directory
		return FilesystemErr{fmt.Errorf("%s is not a directory", s.KeyDir)}
	}
	if info.Mode().Perm() != 0700 {
		// Permissions are wrong: fix 'em
		if err := os.Chmod(s.KeyDir, 0700); err != nil {
			return FilesystemErr{err}
		}
	}

	// Make sure the files we're going to write to don't already exist
	if fileExists(s.privateKeyPath()) {
		return SSHKeysAlreadyExistErr{s.privateKeyPath()}
	}
	if fileExists(s.publicKeyPath()) {
		return SSHKeysAlreadyExistErr{s.privateKeyPath()}
	}

	// The directory looks good as-is
	return nil
}

// WriteKeys writes the SSH key pair to disk.
func (s *SSHKeyPair) WriteKeys() error {
	if len(s.PrivateKeyPEM) == 0 || len(s.PublicKey) == 0 {
		return ErrMissingSSHKeys
	}

	if err := s.PrepFilesystem(); err != nil {
		return err
	}

	if err := writeKeyToFile(s.PrivateKeyPEM, s.privateKeyPath()); err != nil {
		return err
	}
	if err := writeKeyToFile(s.PublicKey, s.publicKeyPath()); err != nil {
		return err
	}

	return nil
}

func writeKeyToFile(keyBytes []byte, path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return ioutil.WriteFile(path, keyBytes, 0600)
	}
	return FilesystemErr{fmt.Errorf("file %s already exists", path)}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}
	return true
}

// attaches a user@host suffix to a serialized public key. returns the original
// pubkey if we can't get the username or host.
func pubKeyWithMemo(pubKey []byte) []byte {
	u, err := user.Current()
	if err != nil {
		return pubKey
	}
	hostname, err := os.Hostname()
	if err != nil {
		return pubKey
	}

	return append(bytes.TrimRight(pubKey, "\n"), []byte(fmt.Sprintf(" %s@%s\n", u.Username, hostname))...)
}
