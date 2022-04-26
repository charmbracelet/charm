package server_test

import (
	"testing"

	"github.com/charmbracelet/charm/testserver"
)

func TestSSHAuthMiddleware(t *testing.T) {
	cl := testserver.SetupTestServer(t)
	auth, err := cl.Auth()
	if err != nil {
		t.Fatalf("auth error: %s", err)
	}
	if auth.JWT == "" {
		t.Fatal("auth error, missing JWT")
	}
	if auth.ID == "" {
		t.Fatal("auth error, missing ID")
	}
	if auth.PublicKey == "" {
		t.Fatal("auth error, missing PublicKey")
	}
	// if len(auth.EncryptKeys) == 0 {
	// 	t.Fatal("auth error, missing EncryptKeys")
	// }
}
