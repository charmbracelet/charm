Charm
=====

Charm is a set of tools that makes adding a backend to your terminal-based
applications fun and easy. Quickly build modern CLI applications without
worrying about user accounts, data storage and encryption.

Charm powers terminal apps like [Glow][skate] and [Skate][skate].

## Features

* [**Charm Accounts:**](#charm-accounts) invisible user account creation and authentication
* [**Charm KV:**](#charm-kv) a fast, cloud-synced key value store built on [BadgerDB][badger]
* [**Charm FS:**](#charm-fs) a Go `fs.FS` compatible cloud-based user filesystem
* **Charm Encrypt:** end-to-end encryption for stored data and on-demand encryption for arbitrary data

By default, applications built with Charm use Charmbracelet, Inc. servers,
however it's also very easy for users to [self-host](#self-hosting).

## Charm Accounts

The best part of Charm accounts is that both you and your users don’t need to
think about them. Charm authentication is based on SSH keys, so account
creation and authentication is built into all Charm tools and is invisible and
frictionless.

If a user already has Charm keys, we authenticate with them. If not, we create
new ones. Users can also easily link multiple machines to their account, and
linked machines will seamlessly gain access to their owners Charm data. Of
course, users can revoke machines’ access too.

## Charm KV

A simple, powerful, cloud-backed key-value store built on [BadgerDB][badger].
Charm KV is a quick, powerful way to get started building apps with Charm.

```go
import "github.com/charmbracelet/charm/kv"

// Open a database (or create one if it doesn’t exist)
db, err := kv.OpenWithDefaults("my-cute-db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Fetch updates (for this user)
db.Sync()

// Save some data (for this user)
if err := db.Set([]byte("fave-food"), []byte("gherkin")); err != nil {
    log.Fatal(err)
}

// All data is binary
if err := db.Set([]byte("profile-pic"), someJPEG); err != nil {
    log.Fatal(err)
}
```

When you use Charm KV, your users get:

* Invisible authentication
* Automatic cloud backup, with the ability to self-host
* Full encryption, both at rest and end-to-end in the cloud
* Syncing across machines

Charm KV can also enhance existing [BadgerDB][badger] implementations. It works
with standard Badger transactions and provides top level functions that mirror
those in Badger.

For details on Charm KV, see [the Charm KV docs][kv].

## Charm FS

A user-based virtual filesystem.

```go
import charmfs "github.com/charmbracelet/charm/fs"

// Open the user’s filesystem
cfs, err := charmfs.NewFS()
if err != nil {
    log.Fatal(err)
}

// Save a file
data := bytes.NewBuffer([]byte("some data"))
if err := cfs.WriteFile("./path/to/file", data, fs.FileMode(0644)); err != nil {
    log.Fatal(err)
}

// Get a file
f, err := cfs.Open("./path/to/file")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

// Just read whole file in one shot
data, err := cfs.ReadFile("./path/to/file")
if err != nil {
    log.Fatal(err)
}
```

Each Charm user has a virtual personal filesystem on the Charm server.  Charm
FS provides a Go [fs.FS](https://golang.org/pkg/io/fs/) implementation for the
user along with additional write and delete methods. If you're a building
a tool that requires file storage, Charm FS will provide it on
a networked-basis without friction-filled authentication flows.

For more on Charm FS see [the Charm FS docs][fs].

## Self Hosting

Charm libraries point at our Charmbracelet, Inc. servers by default (that’s
api.charm.sh), however it's very simple for users to host their own Charm
instances. The `charm` binary is a single, statically-linked executable capable
of serving an entire Charm instance:

```bash
charm serve
```

You can also use the Docker image, which has the benefit of putting the server
behind HTTPS:

```bash
docker pull charm:latest
docker run charm
```

To change hosts users can set `CHARM_HOST` to the domain or IP or their
choosing:

```bash
export CHARM_HOST=burrito.example.com
```


## The Charm Client

The `charm` binary also includes easy access to a lot of the functionality
available in the libraries. This could be useful in scripts, as a standalone
utility or when testing functionality.

```bash
# Link a machine to your Charm account
charm link

# Set a value
charm kv set weather humid

# Print out a tree of your files
charm fs tree /

# Encrypt something
charm encrypt < secretphoto.jpg > encrypted.jpg.json

# For more info
charm help
```

## Projects using Charm

* [Glow][glow]: Render markdown on the CLI, with pizzazz! 💅🏻
* [Skate][skate]: A personal key value store 🛼
* Your app here! Let us know what you build: [vt100@charm.sh](mailto:vt100@charm.sh)

## License

[MIT](https://github.com/charmbracelet/charm/raw/master/LICENSE)

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="the Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source


[releases]: https://github.com/charmbracelet/charm/releases
[docs]: https://pkg.go.dev/github.com/charmbracelet/charm?tab=doc
[kv]: https://github.com/charmbracelet/charm/tree/master/kv
[fs]: https://github.com/charmbracelet/charm/tree/master/fs
[glow]: https://github.com/charmbracelet/glow
[skate]: https://github.com/charmbracelet/skate
[badger]: https://github.com/dgraph-io/badger
