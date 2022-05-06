package server_test

import (
	"testing"

	"github.com/charmbracelet/charm/testserver"
)

func TestHTTPAccess(t *testing.T) {
	cl := testserver.SetupTestServer(t)
	_, err := cl.Auth()
	if err != nil {
		t.Fatalf("auth error: %s", err)
	}

	_, err = cl.AuthedRawRequest("GET", "/v1/fs/../../db/charm_sqlite.db")
	if err == nil {
		t.Fatalf("expected access error, got nil")
	}
}
