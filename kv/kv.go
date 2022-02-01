// Package kv provides a Charm Cloud backed BadgerDB.
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
	"github.com/charmbracelet/charm/fs"
	badger "github.com/dgraph-io/badger/v3"
)

// KV provides a Charm Cloud backed BadgerDB key-value store.
//
// KV supports regular Badger transactions, and backs up the data to the Charm
// Cloud. It will allow for syncing across machines linked with a Charm
// account. All data is encrypted by Badger on the local disk using a Charm
// user's encryption keys. Diffs are also encrypted locally before being synced
// to the Charm Cloud.
type KV struct {
	DB   *badger.DB
	name string
	cc   *client.Client
	fs   *fs.FS
}

// Open a Charm Cloud managed Badger DB instance with badger.Options and
// *client.Client.
func Open(cc *client.Client, name string, opt badger.Options) (*KV, error) {
	db, err := openDB(cc, name, opt)
	if err != nil {
		return nil, err
	}
	fs, err := fs.NewFSWithClient(cc)
	if err != nil {
		return nil, err
	}
	return &KV{DB: db, name: name, cc: cc, fs: fs}, nil
}

// OpenWithDefaults opens a Charm Cloud managed Badger DB instance with the
// default settings pulled from environment variables.
func OpenWithDefaults(name string) (*KV, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return nil, err
	}
	dd, err := client.DataPath(cc.Config.Host)
	if err != nil {
		return nil, err
	}
	pn := filepath.Join(dd, "/kv/", name)
	opts := badger.DefaultOptions(pn).WithLoggingLevel(badger.ERROR)

	// By default we have no logger as it will interfere with Bubble Tea
	// rendering. Use Open with custom options to specify one.
	opts.Logger = nil

	// We default to a 10MB vlog max size (which BadgerDB turns into 20MB vlog
	// files). The Badger default results in 2GB vlog files, which is quite
	// large. This will limit the values to 10MB maximum size. If you need more,
	// please use Open with custom options.
	opts = opts.WithValueLogFileSize(10000000)
	return Open(cc, name, opts)
}

// OptionsWithEncryption returns badger.Options with all required encryption
// settings enabled for a given encryption key.
func OptionsWithEncryption(opt badger.Options, encKey []byte, cacheSize int64) (badger.Options, error) {
	if cacheSize <= 0 {
		return opt, fmt.Errorf("You must set an index cache size to use encrypted workloads in Badger v3")
	}
	return opt.WithEncryptionKey(encKey).WithIndexCacheSize(cacheSize), nil
}

// NewTransaction creates a new *badger.Txn with a Charm Cloud managed
// timestamp.
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

// NewStream returns a new *badger.Stream from the underlying Badger DB.
func (kv *KV) NewStream() *badger.Stream {
	return kv.DB.NewStreamAt(math.MaxUint64)
}

// View wraps the View() method for the underlying Badger DB.
func (kv *KV) View(fn func(txn *badger.Txn) error) error {
	return kv.DB.View(fn)
}

// Sync synchronizes the local Badger DB with any updates from the Charm Cloud.
func (kv *KV) Sync() error {
	return kv.syncFrom(kv.DB.MaxVersion())
}

// Commit commits a *badger.Txn and syncs the diff to the Charm Cloud.
func (kv *KV) Commit(txn *badger.Txn, callback func(error)) error {
	mv := kv.DB.MaxVersion()
	err := kv.syncFrom(mv)
	if err != nil {
		return err
	}
	seq, err := kv.nextSeq(kv.name)
	if err != nil {
		return err
	}
	err = txn.CommitAt(seq, callback)
	if err != nil {
		return err
	}
	return kv.backupSeq(mv, seq)
}

// Close closes the underlying Badger DB.
func (kv *KV) Close() error {
	return kv.DB.Close()
}

// Set is a convenience method for setting a key and value. It creates and
// commits a new transaction for the update.
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

// SetReader is a convenience method to set the value for a key to the data
// read from the provided io.Reader.
func (kv *KV) SetReader(key []byte, value io.Reader) error {
	v, err := ioutil.ReadAll(value)
	if err != nil {
		return err
	}
	return kv.Set(key, v)
}

// Get is a convenience method for getting a value from the key value store.
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

// Delete is a convenience method for deleting a value from the key value store.
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

// Keys returns a list of all keys for this key value store.
func (kv *KV) Keys() ([][]byte, error) {
	var ks [][]byte
	err := kv.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
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

// Client returns the underlying *client.Client.
func (kv *KV) Client() *client.Client {
	return kv.cc
}

// Reset deletes the local copy of the Badger DB and rebuilds with a fresh sync
// from the Charm Cloud
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
