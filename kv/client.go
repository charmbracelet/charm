package kv

import (
	"bytes"
	"fmt"
	"io/fs"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	badger "github.com/dgraph-io/badger/v3"
)

func (kv *KV) seqStorageKey(seq uint64) string {
	return strings.Join([]string{kv.name, fmt.Sprintf("%d", seq)}, "/")
}

func (kv *KV) backupSeq(from uint64, at uint64) error {
	buf := bytes.NewBuffer(nil)
	s := kv.DB.NewStreamAt(math.MaxUint64)
	if _, err := s.Backup(buf, from); err != nil {
		return err
	}
	name := kv.seqStorageKey(at)
	return kv.fs.WriteFile(name, buf.Bytes(), fs.FileMode(0o660))
}

func (kv *KV) restoreSeq(seq uint64) error {
	// there is never a zero seq
	if seq == 0 {
		return nil
	}
	r, err := kv.fs.Open(kv.seqStorageKey(seq))
	if err != nil {
		return err
	}
	defer r.Close() // nolint:errcheck
	// TODO DB.Load() should be called on a database that is not running any
	// other concurrent transactions while it is running.
	return kv.DB.Load(r, 1)
}

func (kv *KV) getSeq(name string) (uint64, error) {
	var sm *charm.SeqMsg
	name, err := kv.fs.EncryptPath(name)
	if err != nil {
		return 0, err
	}
	err = kv.cc.AuthedJSONRequest("GET", fmt.Sprintf("/v1/seq/%s", name), nil, &sm)
	if err != nil {
		return 0, err
	}
	return sm.Seq, nil
}

func (kv *KV) nextSeq(name string) (uint64, error) {
	var sm *charm.SeqMsg
	name, err := kv.fs.EncryptPath(name)
	if err != nil {
		return 0, err
	}
	err = kv.cc.AuthedJSONRequest("POST", fmt.Sprintf("/v1/seq/%s", name), nil, &sm)
	if err != nil {
		return 0, err
	}
	return sm.Seq, nil
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
			err = kv.restoreSeq(i)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func encryptKeyToBadgerKey(k *charm.EncryptKey) ([]byte, error) {
	ek := []byte(k.Key)
	if len(ek) < 32 {
		return nil, fmt.Errorf("encryption key is too short")
	}
	return ek[0:32], nil
}

func openDB(cc *client.Client, opt badger.Options) (*badger.DB, error) {
	var db *badger.DB
	eks, err := cc.EncryptKeys()
	if err != nil {
		return nil, err
	}
	for _, k := range eks {
		ek, err := encryptKeyToBadgerKey(k)
		if err == nil {
			opt, err = OptionsWithEncryption(opt, ek, 32768)
			if err != nil {
				continue
			}
			db, err = badger.OpenManaged(opt)
			if err == nil {
				break
			}
			if err != nil {
				return nil, err
			}
		}
	}
	if db == nil {
		return nil, fmt.Errorf("could not open BadgerDB, bad encrypt keys")
	}
	return db, nil
}
