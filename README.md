Charm
=====

<p>
  <img src="https://stuff.charm.sh/charm/charm-header.png?8" width="317" alt="A little cloud with a pleased expression followed by the words ‘Charm from Charm’"><br>
  <a href="https://github.com/charmbracelet/charm/releases"><img src="https://img.shields.io/github/release/charmbracelet/charm.svg" alt="Latest Release"></a>
  <a href="https://pkg.go.dev/github.com/charmbracelet/charm?tab=doc"><img src="https://godoc.org/github.com/golang/gddo?status.svg" alt="GoDoc"></a>
  <a href="https://github.com/charmbracelet/charm/actions"><img src="https://github.com/charmbracelet/charm/workflows/build/badge.svg" alt="Build Status"></a>
</p>

Charm is a set of tools that makes adding a backend to your terminal-based
applications fun and easy. Quickly build modern CLI applications without
worrying about user accounts, data storage and encryption.

Charm powers terminal apps like [Glow][glow] and [Skate][skate].

## Features

* [**Charm KV:**](#charm-kv) an embeddable, encrypted, cloud-synced key-value store built on [BadgerDB][badger]
* [**Charm FS:**](#charm-fs) a Go `fs.FS` compatible cloud-based user filesystem
* [**Charm Crypt:**](#charm-crypt) end-to-end encryption for stored data and on-demand encryption for arbitrary data
* [**Charm Accounts:**](#charm-accounts) invisible user account creation and authentication

There’s also the powerful [Charm Client](#charm-client) for directly accessing
Charm services. [Self-hosting](#self-hosting) a Charm Cloud is as simple as
running `charm serve`.

## Charm KV

A powerful, embeddable key-value store built on [BadgerDB][badger]. Store user
data, configuration, create a cache or even store large files as values.

When you use Charm KV your users automatically get cloud backup, multi-machine
syncing, end-to-end encryption, and the option to self-host.

```go
import "github.com/charmbracelet/charm/kv"

// Open a database (or create one if it doesn’t exist)
db, err := kv.OpenWithDefaults("my-cute-db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Fetch updates and easily define your own syncing strategy
if err := db.Sync(); err != nil {
    log.Fatal(err)
}

// Save some data
if err := db.Set([]byte("fave-food"), []byte("gherkin")); err != nil {
    log.Fatal(err)
}

// All data is binary
if err := db.Set([]byte("profile-pic"), someJPEG); err != nil {
    log.Fatal(err)
}
```

Charm KV can also enhance existing [BadgerDB][badger] implementations. It works
with standard Badger transactions and provides top level functions that mirror
those in Badger.

For details on Charm KV, see [the Charm KV docs][kv].

## Charm FS

Each Charm user has a virtual personal filesystem on the Charm server. Charm
FS provides a Go [fs.FS](https://golang.org/pkg/io/fs/) implementation for the
user along with additional write and delete methods. If you're a building
a tool that requires file storage, Charm FS will provide it on
a networked-basis without friction-filled authentication flows.

```go
import charmfs "github.com/charmbracelet/charm/fs"

// Open the user’s filesystem
cfs, err := charmfs.NewFS()
if err != nil {
    log.Fatal(err)
}

// Save a file
data := bytes.NewBuffer([]byte("some data"))
if err := cfs.WriteFile("./path/to/file", data, fs.FileMode(0644), int64(data.Len())); err != nil {
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

For more on Charm FS see [the Charm FS docs][fs].

## Charm Crypt

All data sent to a Charm server is fully encrypted on the client. Charm Crypt
provides methods for easily encrypting and decrypting data for a Charm user.
All key management and account linking is handled seamlessly by Charm.

For more on Charm Crypt see [the Charm Crypt docs][crypt].

## Charm Accounts

The best part of Charm accounts is that both you and your users don’t need to
think about them. Charm authentication is based on SSH keys, so account
creation and authentication is built into all Charm tools and is invisible and
frictionless.

If a user already has Charm keys, we authenticate with them. If not, we create
new ones. Users can also easily link multiple machines to their account, and
linked machines will seamlessly gain access to their owners Charm data. Of
course, users can revoke machines’ access too.

## Charm Client

The [`charm`][releases] binary also includes easy access to a lot of the functionality
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

### Installation

Use a package manager:

```bash
# macOS or Linux
brew tap charmbracelet/tap && brew install charmbracelet/tap/charm

# Arch Linux (btw)
yay -S charm-tool

# Nix
nix-env -iA nixpkgs.charm
```

Or download a package or binary from the [releases][releases] page. All
major platforms and architectures are supported, including FreeBSD and ARM.

You can also just build and install it yourself:

```bash
git clone https://github.com/charmbracelet/charm.git
cd charm
go install
```

## Self-Hosting

Charm libraries point at our Charmbracelet, Inc. servers by default (that’s
cloud.charm.sh), however it's very easy for users to host their own Charm
instances. The `charm` binary is a single, statically-linked executable capable
of serving an entire Charm instance:

```bash
charm serve
```

To change hosts users can set `CHARM_HOST` to the domain or IP or their
choosing:

```bash
export CHARM_HOST=burrito.example.com
```

See instructions for [Systemd](https://github.com/charmbracelet/charm/blob/main/systemd.md) and [Docker](https://github.com/charmbracelet/charm/blob/main/docker.md).

### TLS

To set up TLS, you should set `CHARM_SERVER_HTTP_SCHEME` environment variable to
`https` and specify `CHARM_SERVER_HOST`, `CHARM_SERVER_TLS_KEY_FILE`, and
`CHARM_SERVER_TLS_CERT_FILE` file paths.

## Projects using Charm

* [Glow][glow]: Render markdown on the CLI, with pizzazz! 💅🏻
* [Skate][skate]: A personal key-value store 🛼
* Your app here! Let us know what you build: [vt100@charm.sh](mailto:vt100@charm.sh)

## License

[MIT](https://github.com/charmbracelet/charm/raw/main/LICENSE)

***

Part of [Charm](https://charm.sh).

<a href="https://charm.sh/"><img alt="the Charm logo" src="https://stuff.charm.sh/charm-badge-unrounded.jpg" width="400"></a>

Charm热爱开源 • Charm loves open source


[releases]: https://github.com/charmbracelet/charm/releases
[docs]: https://pkg.go.dev/github.com/charmbracelet/charm?tab=doc
[kv]: https://github.com/charmbracelet/charm/tree/main/kv
[fs]: https://github.com/charmbracelet/charm/tree/main/fs
[crypt]: https://github.com/charmbracelet/charm/tree/main/crypt
[glow]: https://github.com/charmbracelet/glow
[skate]: https://github.com/charmbracelet/skate
[badger]: https://github.com/dgraph-io/badger
