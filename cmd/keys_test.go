package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/charm/testserver"
	"github.com/charmbracelet/keygen"
	"golang.org/x/crypto/ssh"
)

func TestKeys(t *testing.T) {
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
		key, err := keygen.New(filepath.Join(t.TempDir(), "test"), nil, keygen.Ed25519)
		if err != nil {
			t.Fatal(err)
		}

		signer, err := ssh.ParsePrivateKey(key.PrivateKeyPEM())
		if err != nil {
			t.Fatal(err)
		}

		cli := testserver.SetupTestServerWithAgent(t, signer)

		if err := cli.NoAgent.LinkKeyToUser(signer.PublicKey()); err != nil {
			t.Fatal(err)
		}

		keys, err := cli.Full.AuthorizedKeysWithMetadata()
		if err != nil {
			t.Fatal(err)
		}

		if l := len(keys.Keys); l != 2 {
			t.Fatalf("expected 2 key, got %d", l)
		}

		t.Run("should keep access with agent after deleting the charm-generated keys", func(t *testing.T) {
			if err := os.RemoveAll(cli.NoAgent.Config.DataDir); err != nil {
				t.Fatal(err)
			}

			if _, err := cli.Full.SetName("foo"); err != nil {
				t.Fatal(err)
			}
		})
	})
}
