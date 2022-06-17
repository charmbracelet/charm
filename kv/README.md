# Charm KV

## Example

```go
package main

import (
	"fmt"

	"github.com/charmbracelet/charm/kv"
	"github.com/dgraph-io/badger/v3"
)

func main() {
	// Open a kv store with the name "charm.sh.test.db" and local path ./db
	db, err := kv.OpenWithDefaults("charm.sh.test.db")
	if err != nil {
		panic(err)
	}

	// Get the latest updates from the Charm Cloud
	db.Sync()

	// Quickly set a value
	err = db.Set([]byte("dog"), []byte("food"))
	if err != nil {
		panic(err)
	}

	// Quickly get a value
	v, err := db.Get([]byte("dog"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("got value: %s\n", string(v))

	// Go full-blown Badger and use transactions to list values and keys
	db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close() //nolint:errcheck
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				fmt.Printf("%s - %s\n", k, v)
				return nil
			})
			if err != nil {
				panic(err)
			}
		}
		return nil
	})
}
```

## Deleting a Database

1. Find the database in `charm fs ls /`
2. Delete the database with `charm fs rm db-name`
3. Locate the local copy of the database. To see where your charm-related data lives, run `charm` to start up with GUI, then select `Backup`
4. Run `rm ~/path/to/cloud.charm.sh/kv/db-name` to remove the local copy of your charm-kv database
