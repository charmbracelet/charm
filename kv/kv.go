package kv

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/charmbracelet/charm/client"
	badger "github.com/dgraph-io/badger/v3"
)

type KV struct {
	DB   *badger.DB
	name string
	cc   *client.Client
}

func Open(cc *client.Client, name string, opt badger.Options) (*KV, error) {
	db, err := openDB(cc, name, opt)
	if err != nil {
		return nil, err
	}
	return &KV{DB: db, name: name, cc: cc}, nil
}

func OpenWithDefaults(name string, path string) (*KV, error) {
	cfg, err := client.ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	cc, err := client.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	pn := filepath.Join(path, name)
	opts := badger.DefaultOptions(pn).WithLoggingLevel(badger.ERROR)
	return Open(cc, name, opts)
}

func OptionsWithEncryption(opt badger.Options, encKey []byte, cacheSize int64) (badger.Options, error) {
	if cacheSize <= 0 {
		return opt, fmt.Errorf("You must set an index cache size to use encrypted workloads in Badger v3")
	}
	return opt.WithEncryptionKey(encKey).WithIndexCacheSize(cacheSize), nil
}

func (kv *KV) NewTransaction(update bool) (*badger.Txn, error) {
	var ts uint64
	var err error
	if update {
		ts, err = kv.getSeq(kv.name)
		if err != nil {
			return nil, err
		}
	} else {
		ts = math.MaxUint64
	}
	return kv.DB.NewTransactionAt(ts, update), nil
}

// func (kv *KV) NewWriteBatch() (*badger.WriteBatch, error) {
// 	seq, err := kv.cc.NextSeq(kv.name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return kv.DB.NewWriteBatchAt(seq), nil
// }

func (kv *KV) NewStream() *badger.Stream {
	return kv.DB.NewStreamAt(math.MaxUint64)
}

// TODO Update
// TODO Load
func (kv *KV) Load() error {
	return fmt.Errorf("not implemented")
}

func (kv *KV) View(fn func(txn *badger.Txn) error) error {
	return kv.DB.View(fn)
}

func (kv *KV) Sync() error {
	mv := kv.DB.MaxVersion()
	seq, err := kv.getSeq(kv.name)
	if err != nil {
		return err
	}
	for i := uint64(mv); i <= seq; i++ {
		err := kv.decryptAndRestoreSeq(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (kv *KV) Commit(txn *badger.Txn, callback func(error)) error {
	mv := kv.DB.MaxVersion()
	seq, err := kv.nextSeq(kv.name)
	if err != nil {
		return err
	}
	for i := uint64(mv); i < seq; i++ {
		err := kv.decryptAndRestoreSeq(i)
		if err != nil {
			return err
		}
	}
	err = txn.CommitAt(seq, callback)
	if err != nil {
		return err
	}
	return kv.encryptAndBackupSeq(mv, seq)
}

func (kv *KV) Close() error {
	return kv.DB.Close()
}

func (kv *KV) Set(key []byte, value []byte) error {
	txn, err := kv.NewTransaction(true)
	if err != nil {
		return err
	}
	err = txn.Set(key, value)
	if err != nil {
		return err
	}
	return kv.Commit(txn, func(err error) {
		if err != nil {
			log.Printf("Badger commit error: %s", err)
		}
	})
}

func (kv *KV) SetReader(key []byte, value io.Reader) error {
	v, err := ioutil.ReadAll(value)
	if err != nil {
		return err
	}
	return kv.Set(key, v)
}

func (kv *KV) Get(key []byte) ([]byte, error) {
	var v []byte
	err := kv.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		v, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return v, err
	}
	return v, nil
}

func (kv *KV) Delete(key []byte) error {
	txn, err := kv.NewTransaction(true)
	if err != nil {
		return err
	}
	err = txn.Delete(key)
	if err != nil {
		return err
	}
	return kv.Commit(txn, func(err error) {
		if err != nil {
			log.Printf("Badger commit error: %s", err)
		}
	})
}

func (kv *KV) Keys() ([][]byte, error) {
	var ks [][]byte
	err := kv.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			ks = append(ks, it.Item().Key())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ks, nil
}

func (kv *KV) Reset() error {
	opts := kv.DB.Opts()
	err := kv.DB.Close()
	if err != nil {
		return err
	}
	err = os.RemoveAll(opts.Dir)
	if err != nil {
		return err
	}
	if opts.ValueDir != opts.Dir {
		err = os.RemoveAll(opts.ValueDir)
		if err != nil {
			return err
		}
	}
	db, err := openDB(kv.cc, kv.name, opts)
	if err != nil {
		return err
	}
	kv.DB = db
	return kv.Sync()
}

func openDB(cc *client.Client, name string, opt badger.Options) (*badger.DB, error) {
	ek, err := encryptKeyFromCharmClient(cc)
	if err != nil {
		return nil, err
	}
	opt, err = OptionsWithEncryption(opt, ek, 32768)
	if err != nil {
		return nil, err
	}
	return badger.OpenManaged(opt)
}
