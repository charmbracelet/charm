package kv

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/charmbracelet/charm/client"
	badger "github.com/dgraph-io/badger/v3"
)

func TestKVSetGet(t *testing.T) {
	pn, err := ioutil.TempDir("", "charmkv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(pn)

	cc, err := client.NewClientWithDefaults()
	if err != nil {
		t.Fatal(err)
	}

	opts := badger.DefaultOptions(pn).WithLoggingLevel(badger.ERROR)
	opts.Logger = nil
	db, err := Open(cc, "test", opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Sync(); err != nil {
		t.Fatal(err)
	}

	key := []byte("foo")
	value := []byte("bar")

	// Save some data
	if err := db.Set(key, value); err != nil {
		t.Fatal(err)
	}

	b, err := db.Get(key)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(value, b) {
		t.Errorf("Expected %s, got %s", value, b)
	}
}
