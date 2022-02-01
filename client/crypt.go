package client

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/google/uuid"
	"github.com/muesli/sasquatch"
)

// KeyForID returns the decrypted EncryptKey for a given key ID.
func (cc *Client) KeyForID(gid string) (*charm.EncryptKey, error) {
	if len(cc.plainTextEncryptKeys) == 0 {
		err := cc.cryptCheck()
		if err != nil {
			return nil, fmt.Errorf("failed crypt check: %w", err)
		}
	}
	if gid == "" {
		if len(cc.plainTextEncryptKeys) == 0 {
			return nil, fmt.Errorf("No keys stored")
		}
		return cc.plainTextEncryptKeys[0], nil
	}
	for _, k := range cc.plainTextEncryptKeys {
		if k.ID == gid {
			return k, nil
		}
	}
	return nil, fmt.Errorf("Key not found for id %s", gid)
}

// DefaultEncryptKey returns the default EncryptKey for an authed user.
func (cc *Client) DefaultEncryptKey() (*charm.EncryptKey, error) {
	return cc.KeyForID("")
}

func (cc *Client) findIdentities() ([]sasquatch.Identity, error) {
	keys, err := FindAuthKeys(cc.Config.Host, cc.Config.KeyType)
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

// EncryptKeys returns all of the symmetric encrypt keys for the authed user.
func (cc *Client) EncryptKeys() ([]*charm.EncryptKey, error) {
	err := cc.cryptCheck()
	if err != nil {
		return nil, err
	}
	return cc.plainTextEncryptKeys, nil
}

func (cc *Client) addEncryptKey(pk string, gid string, key string, createdAt *time.Time) error {
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
	ek.ID = gid
	ek.Key = encKey
	ek.CreatedAt = createdAt

	return cc.AuthedJSONRequest("POST", "/v1/encrypt-key", &ek, nil)
}

func (cc *Client) cryptCheck() error {
	cc.encryptKeyLock.Lock()
	defer cc.encryptKeyLock.Unlock()
	auth, err := cc.Auth()
	if err != nil {
		return err
	}

	if len(auth.EncryptKeys) == 0 && len(cc.plainTextEncryptKeys) == 0 {
		// if there are no encrypt keys, make one for the public key returned from auth
		b := make([]byte, 64)
		_, err := rand.Read(b)
		if err != nil {
			return err
		}
		k := base64.StdEncoding.EncodeToString(b)
		ek := &charm.EncryptKey{}
		ek.PublicKey = auth.PublicKey
		ek.ID = uuid.New().String()
		ek.Key = k
		err = cc.addEncryptKey(ek.PublicKey, ek.ID, ek.Key, nil)
		if err != nil {
			return err
		}
		cc.plainTextEncryptKeys = []*charm.EncryptKey{ek}
		return nil
	}

	if len(auth.EncryptKeys) > len(cc.plainTextEncryptKeys) {
		// if the encryptKeys haven't been decrypted yet, use the sasquatch ids to decrypt them
		sids, err := cc.findIdentities()
		if err != nil {
			return err
		}
		ks := make([]*charm.EncryptKey, 0)
		for _, k := range auth.EncryptKeys {
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
			dk.ID = k.ID
			dk.CreatedAt = k.CreatedAt
			ks = append(ks, dk)
		}
		cc.plainTextEncryptKeys = ks
	}

	return nil
}
