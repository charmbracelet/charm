package charm

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/google/uuid"
	"github.com/muesli/sasquatch"
)

type EncryptKey struct {
	GlobalID  string `json:"global_id"`
	Key       string `json:"key"`
	PublicKey string `json:"public_key,omitempty"`
}

func (cc *Client) Encrypt(content []byte) ([]byte, string, error) {
	err := cc.cryptCheck()
	if err != nil {
		return nil, "", err
	}
	k, err := cc.auth.defaultEncryptKey()
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

func (cc *Client) Decrypt(gid string, content []byte) ([]byte, error) {
	err := cc.cryptCheck()
	k, err := cc.auth.keyforID(gid)
	if err != nil {
		return nil, err
	}
	id, err := sasquatch.NewScryptIdentity(k.Key)
	r, err := sasquatch.Decrypt(bytes.NewReader(content), id)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(r)
}

func (cc *Client) encryptKeys() ([]*EncryptKey, error) {
	err := cc.cryptCheck()
	if err != nil {
		return nil, err
	}
	return cc.auth.EncryptKeys, nil
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

func (au *Auth) keyforID(gid string) (*EncryptKey, error) {
	for _, k := range au.EncryptKeys {
		if k.GlobalID == gid {
			return k, nil
		}
	}
	return nil, fmt.Errorf("Key not found for id %s", gid)
}

func (au *Auth) defaultEncryptKey() (*EncryptKey, error) {
	if len(au.EncryptKeys) == 0 {
		return nil, fmt.Errorf("No keys stored")
	}
	return au.EncryptKeys[0], nil
}
