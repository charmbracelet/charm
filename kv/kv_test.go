package kv

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	badger "github.com/dgraph-io/badger/v3"
)

func setup(t *testing.T) *badger.DB {
	t.Helper()
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Fatal(err)
	}
	t.Cleanup(func() {
		db.DropAll()
	})
	return db
}

// TestGetForEmptyDB should return an error as no values exist in DB
func TestGetForEmptyDB(t *testing.T) {
	db := setup(t)
	defer db.Close()
	kv := &KV{DB: db, name: "database"}
	_, err := kv.Get([]byte("1234"))
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestGetForValidValue(t *testing.T) {
	db := setup(t)
	defer db.Close()
	want := []byte("yes")
	// Start a writable transaction.
	txn := db.NewTransaction(true)
	defer txn.Discard()

	// Use the transaction...
	err := txn.Set([]byte("1234"), []byte("yes"))
	if err != nil {
		t.Errorf("unable to set kv pair for badgerdb: %v", err)
	}

	// Commit the transaction and check for error.
	if err := txn.Commit(); err != nil {
		t.Errorf("unable to commit kv pair to badgerdb: %v", err)
	}
	kv := &KV{DB: db, name: "database"}
	got, err := kv.Get([]byte("1234"))
	if bytes.Compare(got, want) == 0 {
		t.Errorf("got %s, want %s", got, want)
	}
}
