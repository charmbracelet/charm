package client

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/google/uuid"
	"github.com/muesli/sasquatch"
)

// Encrypt encrypts bytes with the default encrypt key, returning the encrypted
// bytes, encrypt key ID and error.
func (cc *Client) Encrypt(content []byte) ([]byte, string, error) {
	return cc.EncryptWithKey("", content)
}

// EncryptWithKey encrypts bytes with a given encrypt key ID, returning the
// encrypted bytes, encrypt key ID and error.
func (cc *Client) EncryptWithKey(id string, content []byte) ([]byte, string, error) {
	err := cc.cryptCheck()
	if err != nil {
		return nil, "", err
	}
	k, err := cc.keyForID(id)
	if err != nil {
		return nil, "", err
	}

	buf := bytes.NewBuffer(nil)
	r, err := sasquatch.NewScryptRecipient(k.Key)
	if err != nil {
		return nil, "", err
	}
	w, err := sasquatch.Encrypt(buf, r)
	if err != nil {
		return nil, "", err
	}
	w.Write(content)
	w.Close()

	return buf.Bytes(), k.GlobalID, nil
}

// Decrypt decrypts bytes with a given encrypt key ID.
func (cc *Client) Decrypt(gid string, content []byte) ([]byte, error) {
	err := cc.cryptCheck()
	if err != nil {
		return nil, err
	}
	k, err := cc.keyForID(gid)
	if err != nil {
		return nil, err
	}

	id, err := sasquatch.NewScryptIdentity(k.Key)
	if err != nil {
		return nil, err
	}

	r, err := sasquatch.Decrypt(bytes.NewReader(content), id)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(r)
}

func (cc *Client) encryptKeys() ([]*charm.EncryptKey, error) {
	err := cc.cryptCheck()
	if err != nil {
		return nil, err
	}
	return cc.plainTextEncryptKeys, nil
}

func (cc *Client) addEncryptKey(pk string, gid string, key string) error {
	buf := bytes.NewBuffer(nil)
	r, err := sasquatch.ParseRecipient(pk)
	if err != nil {
		return err
	}
	w, err := sasquatch.Encrypt(buf, r)
	if err != nil {
		return err
	}
	w.Write([]byte(key))
	w.Close()

	encKey := base64.StdEncoding.EncodeToString(buf.Bytes())
	ek := charm.EncryptKey{}
	ek.PublicKey = pk
	ek.GlobalID = gid
	ek.Key = encKey

	return cc.AuthedRequest("POST", "/v1/encrypt-key", &ek, nil)
}

func (cc *Client) findIdentities() ([]sasquatch.Identity, error) {
	keys, err := findAuthKeys()
	if err != nil {
		return nil, err
	}

	var ids []sasquatch.Identity
	for _, v := range keys {
		id, err := sasquatch.ParseIdentitiesFile(v)
		if err == nil {
			ids = append(ids, id...)
		}
	}
	return ids, nil
}

func (cc *Client) cryptCheck() error {
	cc.encryptKeyLock.Lock()
	defer cc.encryptKeyLock.Unlock()
	auth, err := cc.Auth()
	if err != nil {
		return err
	}

	if len(cc.auth.EncryptKeys) == 0 && len(cc.plainTextEncryptKeys) == 0 {
		// if there are no encrypt keys, make one for the public key returned from auth
		b := make([]byte, 64)
		_, err := rand.Read(b)
		if err != nil {
			return err
		}
		k := base64.StdEncoding.EncodeToString(b)
		ek := &charm.EncryptKey{}
		ek.PublicKey = auth.PublicKey
		ek.GlobalID = uuid.New().String()
		ek.Key = k
		err = cc.addEncryptKey(ek.PublicKey, ek.GlobalID, ek.Key)
		if err != nil {
			return err
		}
		cc.plainTextEncryptKeys = []*charm.EncryptKey{ek}

		return nil
	}

	if len(cc.auth.EncryptKeys) > len(cc.plainTextEncryptKeys) {
		// if the encryptKeys haven't been decrypted yet, use the sasquatch ids to decrypt them
		sids, err := cc.findIdentities()
		if err != nil {
			return err
		}
		ks := make([]*charm.EncryptKey, 0)
		for _, k := range cc.auth.EncryptKeys {
			ds, err := base64.StdEncoding.DecodeString(k.Key)
			if err != nil {
				return err
			}
			dr, err := sasquatch.Decrypt(bytes.NewReader(ds), sids...)
			if err != nil {
				return err
			}

			buf := new(strings.Builder)
			_, err = io.Copy(buf, dr)
			if err != nil {
				return err
			}

			dk := &charm.EncryptKey{}
			dk.Key = buf.String()
			dk.PublicKey = k.PublicKey
			dk.GlobalID = k.GlobalID
			ks = append(ks, dk)
		}
		cc.plainTextEncryptKeys = ks
	}

	return nil
}

func (cc *Client) keyForID(gid string) (*charm.EncryptKey, error) {
	cc.encryptKeyLock.Lock()
	defer cc.encryptKeyLock.Unlock()
	if gid == "" {
		if len(cc.plainTextEncryptKeys) == 0 {
			return nil, fmt.Errorf("No keys stored")
		}
		return cc.plainTextEncryptKeys[0], nil
	}
	for _, k := range cc.plainTextEncryptKeys {
		if k.GlobalID == gid {
			return k, nil
		}
	}

	return nil, fmt.Errorf("Key not found for id %s", gid)
}
