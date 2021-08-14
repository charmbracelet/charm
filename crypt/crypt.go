// Package crypt provides encryption writer/readers.
package crypt

import (
	"encoding/hex"
	"io"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	"github.com/jacobsa/crypto/siv"
	"github.com/muesli/sasquatch"
)

// Crypt manages the account and encryption keys used for encrypting and
// decrypting.
type Crypt struct {
	key *charm.EncryptKey
}

// EncryptedWriter is an io.WriteCloser. All data written to this writer is
// encrypted before being written to the underlying io.Writer.
type EncryptedWriter struct {
	w io.WriteCloser
}

// DecryptedReader is an io.Reader that decrypts data from an encrypted
// underlying io.Reader.
type DecryptedReader struct {
	r io.Reader
}

// NewCrypt authenticates a user to the Charm Cloud and returns a Crypt struct
// ready for encrypting and decrypting.
func NewCrypt() (*Crypt, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return nil, err
	}
	ek, err := cc.DefaultEncryptKey()
	if err != nil {
		return nil, err
	}
	return NewCryptWithKey(ek), nil
}

// NewCryptWithKey creates a new Crypt with a specific EncryptKey.
func NewCryptWithKey(ek *charm.EncryptKey) *Crypt {
	return &Crypt{key: ek}
}

// NewDecryptedReader creates a new Reader that will read from and decrypt the
// passed in io.Reader of encrypted data.
func (cr *Crypt) NewDecryptedReader(r io.Reader) (*DecryptedReader, error) {
	dr := &DecryptedReader{}
	id, err := sasquatch.NewScryptIdentity(cr.key.Key)
	if err != nil {
		return nil, err
	}
	sdr, err := sasquatch.Decrypt(r, id)
	if err != nil {
		return nil, err
	}
	dr.r = sdr
	return dr, nil
}

// NewEncryptedWriter creates a new Writer that encrypts all data and writes
// the encrypted data to the supplied io.Writer.
func (cr *Crypt) NewEncryptedWriter(w io.Writer) (*EncryptedWriter, error) {
	ew := &EncryptedWriter{}
	rec, err := sasquatch.NewScryptRecipient(cr.key.Key)
	if err != nil {
		return ew, err
	}
	sew, err := sasquatch.Encrypt(w, rec)
	if err != nil {
		return ew, err
	}
	ew.w = sew
	return ew, nil
}

// Key returns the EncryptKey this Crypt is using.
func (cr *Crypt) Key() *charm.EncryptKey {
	return cr.key
}

// EncryptLookupField will deterministically encrypt a string and the same
// encrypted value every time this string is encrypted with the same
// EncryptKey. This is useful if you need to look up an encrypted value without
// knowing the plaintext on the storage side. For writing encrypted data, use
// EncrytpedWriter which is non-deterministic.
func (cr *Crypt) EncryptLookupField(field string) (string, error) {
	if field == "" {
		return "", nil
	}
	ct, err := siv.Encrypt(nil, []byte(cr.key.Key[:32]), []byte(field), nil)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ct), nil
}

// DecryptLookupField decrypts a string encrypted with EncryptLookupField.
func (cr *Crypt) DecryptLookupField(field string) (string, error) {
	if field == "" {
		return "", nil
	}
	ct, err := hex.DecodeString(field)
	if err != nil {
		return "", err
	}
	pt, err := siv.Decrypt([]byte(cr.key.Key[:32]), ct, nil)
	return string(pt), nil
}

// Read decrypts and reads data from the underlying io.Reader.
func (dr *DecryptedReader) Read(p []byte) (int, error) {
	return dr.r.Read(p)
}

// Write encrypts data and writes it to the underlying io.WriteCloser.
func (ew *EncryptedWriter) Write(p []byte) (int, error) {
	return ew.w.Write(p)
}

// Close closes the underlying io.WriteCloser.
func (ew *EncryptedWriter) Close() error {
	return ew.w.Close()
}
