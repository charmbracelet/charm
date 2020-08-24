package keygen

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestSSHKeyGeneration(t *testing.T) {
	var k = &SSHKeyPair{}

	// Create temp directory for keys
	dir, err := ioutil.TempDir("", "charm")
	if err != nil {
		t.Fatalf("error creating temp directory (%s): %v\n", dir, err)
	}

	// Cleanup temp directory after testing
	// NOTE: this is Go 1.14+ only
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("error cleaning up temp directory (%s): %v\n", dir, err)
		}
	}()

	t.Run("test generate SSH keys", func(t *testing.T) {
		err = k.GenerateEd25519Keys()
		if err != nil {
			t.Errorf("error creating SSH key pair: %v", err)
		}

		// TODO: is there a good way to validate these? Lengths seem to vary a bit,
		// so far now we're just asserting that the keys indeed exist.
		if len(k.PrivateKeyPEM) == 0 {
			t.Error("error creating SSH private key PEM; key is 0 bytes")
		}
		if len(k.PublicKey) == 0 {
			t.Error("error creating SSH public key; key is 0 bytes")
		}
	})

	t.Run("test write SSH keys", func(t *testing.T) {
		k.KeyDir = filepath.Join(dir, "ssh1")
		if err := k.PrepFilesystem(); err != nil {
			t.Errorf("filesystem error: %v\n", err)
		}
		if err := k.WriteKeys(); err != nil {
			t.Errorf("error writing SSH keys to %s: %v", k.KeyDir, err)
		}
		if testing.Verbose() {
			t.Logf("Wrote keys to %s", k.KeyDir)
		}
	})

	t.Run("test not overwriting existing keys", func(t *testing.T) {
		k.KeyDir = filepath.Join(dir, "ssh2")
		if err := k.PrepFilesystem(); err != nil {
			t.Errorf("filesystem error: %v\n", err)
		}

		// Private key
		filePath := filepath.Join(k.KeyDir, k.Filename)
		if !touchTestFile(t, filePath) {
			return
		}
		if err := k.WriteKeys(); err == nil {
			t.Errorf("we wrote the private key over an existing file, but we were not supposed to")
		}
		if err := os.Remove(filePath); err != nil {
			t.Errorf("could not remove file %s", filePath)
		}

		// Public key
		if !touchTestFile(t, filePath+".pub") {
			return
		}
		if err := k.WriteKeys(); err == nil {
			t.Errorf("we wrote the public key over an existing file, but we were not supposed to")
		}
	})
}

// touchTestFile is a utility function we're using in testing.
func touchTestFile(t *testing.T, path string) (ok bool) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Errorf("could not create directory %s: %v", dir, err)
		return false
	}
	if _, err := os.Create(path); err != nil {
		t.Errorf("could not create file %s", path)
		return false
	}
	if testing.Verbose() {
		t.Logf("created dummy file at %s", path)
	}
	return true
}
