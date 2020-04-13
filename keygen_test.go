package charm

import (
	"testing"
)

func TestGenerateSSHKeys(t *testing.T) {
	k, err := NewSSHKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	// TODO: is there a good way to validate these? Lengths seem to vary a bit,
	// so far now we're just asserting that the keys indeed exist.
	if len(k.PrivateKeyPEM) == 0 {
		t.Error("error creating SSH private key PEM; key is 0 bytes")
	}
	if len(k.PublicKey) == 0 {
		t.Error("error creating SSH public key; key is 0 bytes")
	}
}
