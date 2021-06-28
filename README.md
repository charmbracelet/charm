Charm
=====

Charm is a set of tools designed to make building Terminal based applications
fun and easy! Quickly build modern CLI applications without worrying about user
accounts, data storage or encryption.

Charm powers the back-end to Terminal apps like
[Glow](https://github.com/charmbracelet/glow) and
[Skate](https://github.com/charmbracelet/skate).

## Features

* Invisible user account creation and authentication
* Golang `fs.FS` compatible cloud based user data storage
* Charm managed and cloud synced BadgerDB key values stores
* End-to-end encryption and user encryption library
* Charmbracelet Inc. hosted default server, with the option for self-hosting

## Charm Accounts

Typical account systems put a lot of burden on users. Who wants to create a new
account before they can try out a piece of software? Charm accounts are based
on SSH keys and account creation is invisible to the user. If they already have
Charm SSH keys, we authenticate with them, if not we'll create a new Charm
account and SSH keys. Users can easily link multiple machines, meaning that any
app built by Charm will seamlessly access that user's account after they link a
new machine. When you use any of the Charm libaries, the user account system
will be handled invisibly for you.

## Charm KV / BadgerDB

The quickest way to get started building apps with Charm is to use our key
value store. Charm provides a managed BadgerDB that's simple to develop with
and transparent to your users. When you use the Charm BadgerDB, your users will
get:

* Cloud backup, with the ability to self-host
* Full encryption, both at rest and end-to-end in the cloud
* Syncing across machines

The [Charm KV](https://github.com/charmbracelet/charm/kv) library makes it easy
to enhance existing BadgerDB implementations. It works with standard Badger
transactions and provides top level functions that mirror those in Badger.

### Example

```go
package main

import (
	"fmt"

	"github.com/charmbracelet/charm/kv"
	"github.com/dgraph-io/badger/v3"
)

func main() {
	// Open a kv store with the name "charm.sh.test.db" and local path ./db
	db, err := kv.OpenWithDefaults("charm.sh.test.db", "./db")
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
		defer it.Close()
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

## Charm FS

Each Charm user has a virtual filesystem on the Charm server. [Charm FS](/fs)
provides a Golang [fs.FS](https://golang.org/pkg/io/fs/) implementation for the
user along with additional write and delete methods. If you're a building a
tool that requires file storage, Charm FS will provide it without
friction-filled authentication flows.

### Example

```go
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

	charmfs "github.com/charmbracelet/charm/fs"
)

func main() {
	// Open the file system
	cfs, err := charmfs.NewFS()
	if err != nil {
		panic(err)
	}

	// Write a file
	data := []byte("some data")
	buf := bytes.NewBuffer(data)
	err = cfs.WriteFile("/our/test/data", buf, fs.FileMode(0644))
	if err != nil {
		panic(err)
	}

	// Get a file
	f, err := cfs.Open("/our/test/data")
	if err != nil {
		panic(err)
	}
	buf = bytes.NewBuffer(nil)
	_, err = io.Copy(buf, f)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(buf.Bytes()))

	// Or use fs.ReadFileFS
	bs, err := cfs.ReadFile("/our/test/data")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(bs))

	// Since we're using fs.FS interfaces we can also do things like walk a tree
	err = fs.WalkDir(cfs, "/", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return nil
	})
	if err != nil {
		panic(err)
	}
}
```

## Charm Server

By default the Charm libraries point at our hosted Charm Cloud (api.charm.sh).
By running `charm serve` and setting the `CHARM_HOST` environment variable,
users can easily self-host their own Charm Cloud.

## Charm Client

The `charm` binary includes easy access to a lot of the functionality available
in the libraries. This could be useful in scripts, as a standalone utility or
when testing functionality. To access the key value store, check out the `charm
kv` commands, `charm fs` for the file store and `charm crypt` for encryption.
The `charm` tool can also be used to link accounts.

## Charming Projects

* [Glow](https://github.com/charmbracelet/glow), Render markdown on the CLI, with pizzazz! 💅🏻
* [Skate](https://github.com/charmbracelet/skate), A personal key value store 🛼
* Your app here! Just let us know what you build at vt100@charm.sh.

## License

[MIT](https://github.com/charmbracelet/charm/raw/master/LICENSE)

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="the Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源! / Charm loves open source!

[releases]: https://github.com/charmbracelet/charm/releases
[docs]: https://pkg.go.dev/github.com/charmbracelet/charm?tab=doc
[glow]: https://github.com/charmbracelet/glow
