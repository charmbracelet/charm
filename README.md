Charm
=====

Charm is a set of tools designed to make building Terminal based applications
fun and easy! Quickly build modern CLI applications without worrying about user
accounts, data storage or encryption.

Charm powers the back-end to Terminal apps like
[https://github.com/charmbracelet/glow](Glow) and
[https://github.com/charmbracelet/skate](Skate).

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

## Charm FS

Each Charm user has a virtual filesystem on the Charm server. [Charm FS](/fs)
provides a Golang [fs.FS](https://golang.org/pkg/io/fs/) implementation for the
user along with additional write and delete methods. If you're a building a
tool that requires file storage, Charm FS will provide it without
friction-filled authentication flows.

```
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

	charmfs "github.com/charmbracelet/charm/fs"
)

func main() {
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

	// Since we're using fs.FS interfaces we can also do typical things, like print a tree
	err = fs.WalkDir(cfs, "/", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return nil
	})
	if err != nil {
		panic(err)
	}

}
```

## Charm KV / BadgerDB

## Charm Crypt

## Charm Server

## Charm Client

## License

[MIT](https://github.com/charmbracelet/charm/raw/master/LICENSE)

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="the Charm logo" src="https://stuff.charm.sh/charm-badge.jpg" width="400"></a>

Charm热爱开源! / Charm loves open source!

[releases]: https://github.com/charmbracelet/charm/releases
[docs]: https://pkg.go.dev/github.com/charmbracelet/charm?tab=doc
[glow]: https://github.com/charmbracelet/glow
