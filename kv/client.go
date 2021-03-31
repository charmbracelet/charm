package kv

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
)

func encryptKeyFromCharmClient(cc *client.Client) ([]byte, error) {
	var auth *client.Auth
	var err error
	auth, err = cc.Auth()
	if err != nil {
		return nil, err
	}
	if len(auth.EncryptKeys) == 0 {
		err = cc.GenerateEncryptKeys()
		if err != nil {
			return nil, err
		}
		auth, err = cc.Auth()
		if err != nil {
			return nil, err
		}
	}
	ek := []byte(auth.EncryptKeys[0].Key)
	if len(ek) < 32 {
		return nil, fmt.Errorf("Encryption key is too short")
	}
	return []byte(ek)[0:32], nil
}

func (kv *KV) seqStorageKey(seq uint64) string {
	return strings.Join([]string{kv.name, fmt.Sprintf("%d", seq)}, "/")
}

func (kv *KV) encryptAndBackupSeq(from uint64, at uint64) error {
	buf := bytes.NewBuffer(nil)
	ew, err := kv.cc.NewEncryptedWriter("", buf)
	s := kv.DB.NewStreamAt(math.MaxUint64)
	_, err = s.Backup(ew, from)
	if err != nil {
		return err
	}
	ew.Close()
	return kv.storeDatalog(kv.seqStorageKey(at), buf.Bytes())
}

func (kv *KV) decryptAndRestoreSeq(seq uint64) error {
	// there is never a zero seq
	if seq == 0 {
		return nil
	}
	r, err := kv.getDatalog(kv.seqStorageKey(seq))
	defer r.Close()
	if err != nil {
		return err
	}
	dr, err := kv.cc.NewDecryptedReader("", r)
	if err != nil {
		return err
	}
	// TODO DB.Load() should be called on a database that is not running any
	// other concurrent transactions while it is running.
	return kv.DB.Load(dr, 1)
}

func (kv *KV) getSeq(name string) (uint64, error) {
	var sm *charm.SeqMsg
	err := kv.cc.AuthedRequest("GET", fmt.Sprintf("/v1/seq/%s", name), nil, &sm)
	if err != nil {
		return 0, err
	}
	return sm.Seq, nil
}

func (kv *KV) nextSeq(name string) (uint64, error) {
	var sm *charm.SeqMsg
	err := kv.cc.AuthedRequest("POST", fmt.Sprintf("/v1/seq/%s", name), nil, &sm)
	if err != nil {
		return 0, err
	}
	return sm.Seq, nil
}

func (kv *KV) storeDatalog(key string, data []byte) error {
	err := kv.fs.WriteFile(key, data, 0)
	if err != nil {
		return err
	}
	return nil
}

func (kv *KV) getDatalog(key string) (io.ReadCloser, error) {
	return kv.fs.Open(key)
}

func (kv *KV) syncFrom(mv uint64) error {
	seqDir, err := kv.fs.ReadDir(kv.name)
	if err != nil {
		return err
	}
	for _, de := range seqDir {
		ii, err := strconv.Atoi(de.Name())
		if err != nil {
			return err
		}
		i := uint64(ii)
		if i > mv {
			err = kv.decryptAndRestoreSeq(i)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
