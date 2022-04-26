package cmd

import (
	"path/filepath"
	"testing"

	"github.com/charmbracelet/charm/testserver"
	"github.com/charmbracelet/keygen"
)

func TestKeys(t *testing.T) {
	/**
	test cases:
	- setup account should create a new local key
	- setup account using agent should create a new local key anyway
	- add a key to an existing account
	- create account and add a key to it
	- add key from agent into existing account
	- add key from agent into new account
	**/

	t.Run("create account", func(t *testing.T) {
		cli := testserver.SetupTestServer(t)
		KeysCmd.SetArgs([]string{"-s"})
		if err := KeysCmd.Execute(); err != nil {
			t.Fatal(err)
		}

		keys, err := cli.AuthorizedKeysWithMetadata()
		if err != nil {
			t.Fatal(err)
		}

		if l := len(keys.Keys); l != 1 {
			t.Fatalf("expected 1 key, got %d", l)
		}
	})

	t.Run("create account and add existing key later", func(t *testing.T) {
		cli := testserver.SetupTestServer(t)

		KeysCmd.SetArgs([]string{"-s"})
		if err := KeysCmd.Execute(); err != nil {
			t.Fatal(err)
		}

		key, err := keygen.New(filepath.Join(t.TempDir(), "test"), nil, keygen.Ed25519)
		if err != nil {
			t.Fatal(err)
		}

		KeysCmd.SetArgs([]string{"-a", string(key.PublicKey())})
		if err := KeysCmd.Execute(); err != nil {
			t.Fatal(err)
		}

		keys, err := cli.AuthorizedKeysWithMetadata()
		if err != nil {
			t.Fatal(err)
		}

		if l := len(keys.Keys); l != 2 {
			t.Fatalf("expected 2 key, got %d", l)
		}
	})

	t.Run("create account adding existing key", func(t *testing.T) {
		cli := testserver.SetupTestServer(t)

		key, err := keygen.New(filepath.Join(t.TempDir(), "test"), nil, keygen.Ed25519)
		if err != nil {
			t.Fatal(err)
		}

		KeysCmd.SetArgs([]string{"-a", string(key.PublicKey())})
		if err := KeysCmd.Execute(); err != nil {
			t.Fatal(err)
		}

		keys, err := cli.AuthorizedKeysWithMetadata()
		if err != nil {
			t.Fatal(err)
		}

		if l := len(keys.Keys); l != 2 {
			t.Fatalf("expected 2 key, got %d", l)
		}
	})

	t.Run("create account adding existing key from ssh agent", func(t *testing.T) {
		// TODO: figure out how to run serve a "test agent"
		// TODO: if agent setting is set, should we add its keys to the user account?
	})
}
