package client

import (
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestAlgo(t *testing.T) {
	for k, v := range map[string]string{
		ssh.KeyAlgoRSA:        "rsa",
		ssh.KeyAlgoDSA:        "dss",
		ssh.KeyAlgoECDSA256:   "ecdsa",
		ssh.KeyAlgoSKECDSA256: "ecdsa",
		ssh.KeyAlgoECDSA384:   "ecdsa",
		ssh.KeyAlgoECDSA521:   "ecdsa",
		ssh.KeyAlgoED25519:    "ed25519",
		ssh.KeyAlgoSKED25519:  "ed25519",
	} {
		t.Run(k, func(t *testing.T) {
			got := algo(k)
			if got != v {
				t.Errorf("expected %q, got %q", v, got)
			}
		})
	}
}

func TestBitsize(t *testing.T) {
	for k, v := range map[string]int{
		ssh.KeyAlgoRSA:        3071,
		ssh.KeyAlgoDSA:        1024,
		ssh.KeyAlgoECDSA256:   256,
		ssh.KeyAlgoSKECDSA256: 256,
		ssh.KeyAlgoECDSA384:   384,
		ssh.KeyAlgoECDSA521:   521,
		ssh.KeyAlgoED25519:    256,
		ssh.KeyAlgoSKED25519:  256,
	} {
		t.Run(k, func(t *testing.T) {
			got := bitsize(k)
			if got != v {
				t.Errorf("expected %d, got %d", v, got)
			}
		})
	}
}
