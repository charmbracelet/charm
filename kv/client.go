package kv

import (
	"bytes"
	"fmt"
	"io/fs"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/charm/client"
	charm "github.com/charmbracelet/charm/proto"
	badger "github.com/dgraph-io/badger/v3"
)

type kvFile struct {
	data *bytes.Buffer
	info *kvFileInfo
}

type kvFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	isDir   bool
	modTime time.Time
}

func (f *kvFileInfo) Name() string {
	return f.name
}

func (f *kvFileInfo) Size() int64 {
	return f.size
}

func (f *kvFileInfo) Mode() fs.FileMode {
	return f.mode
}

func (f *kvFileInfo) ModTime() time.Time {
	return f.modTime
}

func (f *kvFileInfo) IsDir() bool {
	return f.mode&fs.ModeDir != 0
}

func (f *kvFileInfo) Sys() interface{} {
	return nil
}

func (f *kvFile) Stat() (fs.FileInfo, error) {
	if f.info == nil {
		return nil, fmt.Errorf("File info not set")
	}
	return f.info, nil
}

func (f *kvFile) Close() error {
	return nil
}

func (f *kvFile) Read(p []byte) (n int, err error) {
	return f.data.Read(p)
}

func (kv *KV) seqStorageKey(seq uint64) string {
	return strings.Join([]string{kv.name, fmt.Sprintf("%d", seq)}, "/")
}

func (kv *KV) backupSeq(from uint64, at uint64) error {
	buf := bytes.NewBuffer(nil)
	s := kv.DB.NewStreamAt(math.MaxUint64)
	size, err := s.Backup(buf, from)
	if err != nil {
		return err
	}
	name := kv.seqStorageKey(at)
	src := &kvFile{
		data: buf,
		info: &kvFileInfo{
			name:    name,
			size:    int64(size),
			mode:    fs.FileMode(0660),
			modTime: time.Now(),
		},
	}
	return kv.fs.WriteFile(name, src)
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
	defer r.Close()
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
		return nil, fmt.Errorf("Encryption key is too short")
	}
	return []byte(ek)[0:32], nil
}

func openDB(cc *client.Client, name string, opt badger.Options) (*badger.DB, error) {
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
		}
	}
	if db == nil {
		return nil, fmt.Errorf("could not open BadgerDB, bad encrypt keys")
	}
	return db, nil
}
