package charm

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/muesli/sasquatch"
)

type EncryptKey struct {
	GlobalID  string `json:"global_id"`
	Key       string `json:"key"`
	PublicKey string `json:"public_key,omitempty"`
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
	encKey := string(buf.Bytes())

	ek := EncryptKey{}
	ek.PublicKey = pk
	ek.GlobalID = gid
	ek.Key = encKey

	return cc.AuthedRequest("POST", cc.config.BioHost, cc.config.BioPort, "/v1/encrypt-key", &ek, nil)
}

func (cc *Client) cryptCheck() error {
	auth, err := cc.Auth()
	if err != nil {
		return err
	}
	cc.authLock.Lock()
	defer cc.authLock.Unlock()
	if len(cc.auth.EncryptKeys) == 0 {
		// if there are no encrypt keys, make one for the public key returned from auth
		b := make([]byte, 64)
		_, err := rand.Read(b)
		if err != nil {
			return err
		}
		k := base64.StdEncoding.EncodeToString(b)
		ek := &EncryptKey{}
		ek.PublicKey = auth.PublicKey
		ek.GlobalID = uuid.New().String()
		ek.Key = k
		err = cc.addEncryptKey(ek.PublicKey, ek.GlobalID, ek.Key)
		if err != nil {
			return err
		}
		cc.auth.EncryptKeys = []*EncryptKey{ek}
		cc.auth.encryptKeysDecrypted = true
	}
	if cc.auth.encryptKeysDecrypted == false {
		// if the encryptKeys haven't been decrypted yet, use the sasquatch ids to decrypt them
		sids := sasquatch.FindIdentities()
		ks := make([]*EncryptKey, 0)
		for _, k := range cc.auth.EncryptKeys {
			dr, err := sasquatch.Decrypt(strings.NewReader(k.Key), sids...)
			if err != nil {
				return err
			}
			buf := new(strings.Builder)
			_, err = io.Copy(buf, dr)
			if err != nil {
				return err
			}
			dk := &EncryptKey{}
			dk.Key = buf.String()
			dk.PublicKey = k.PublicKey
			dk.GlobalID = k.GlobalID
			ks = append(ks, dk)
		}
		cc.auth.EncryptKeys = ks
		cc.auth.encryptKeysDecrypted = true
	}
	return nil
}

func (cc *Client) Encrypt(content []byte) ([]byte, string, error) {
	err := cc.cryptCheck()
	if err != nil {
		return nil, "", err
	}
	// TODO encrypt content
	return content, cc.auth.EncryptKeys[0].GlobalID, nil
}

// func (cc *Client) decrypt(gid uuid.UUID, content []byte) ([]byte, error) {
// }
