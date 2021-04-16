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
)

func (kv *KV) seqStorageKey(seq uint64) string {
	return strings.Join([]string{kv.name, fmt.Sprintf("%d", seq)}, "/")
}

func (kv *KV) backupSeq(from uint64, at uint64) error {
	buf := bytes.NewBuffer(nil)
	s := kv.DB.NewStreamAt(math.MaxUint64)
	_, err := s.Backup(buf, from)
	if err != nil {
		return err
	}
	return kv.fs.WriteFile(kv.seqStorageKey(at), buf, fs.FileMode(0660))
}

func (kv *KV) restoreSeq(seq uint64) error {
	// there is never a zero seq
	if seq == 0 {
		return nil
	}
	r, err := kv.fs.Open(kv.seqStorageKey(seq))
	defer r.Close()
	if err != nil {
		return err
	}
	// TODO DB.Load() should be called on a database that is not running any
	// other concurrent transactions while it is running.
	return kv.DB.Load(r, 1)
}

func (kv *KV) getSeq(name string) (uint64, error) {
	var sm *charm.SeqMsg
	err := kv.cc.AuthedJSONRequest("GET", fmt.Sprintf("/v1/seq/%s", name), nil, &sm)
	if err != nil {
		return 0, err
	}
	return sm.Seq, nil
}

func (kv *KV) nextSeq(name string) (uint64, error) {
	var sm *charm.SeqMsg
	err := kv.cc.AuthedJSONRequest("POST", fmt.Sprintf("/v1/seq/%s", name), nil, &sm)
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

func encryptKeyFromCharmClient(cc *client.Client) ([]byte, error) {
	k, err := cc.DefaultEncryptKey()
	if err != nil {
		return nil, err
	}
	ek := []byte(k.Key)
	if len(ek) < 32 {
		return nil, fmt.Errorf("Encryption key is too short")
	}
	return []byte(ek)[0:32], nil
}
