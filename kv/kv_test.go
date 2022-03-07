package kv

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/charmbracelet/charm/client"
	badger "github.com/dgraph-io/badger/v3"
)

func setup(t *testing.T) *KV {
	t.Helper()
	opt := badger.DefaultOptions("").WithInMemory(true)
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		log.Fatal(err)
	}
	kv, err := Open(cc, "test", opt)
	if err != nil {
		log.Fatal(err)
	}
	return kv
}

// TestGet
func TestGetForEmptyDB(t *testing.T) {
	kv := setup(t)
	_, err := kv.Get([]byte("1234"))
	if err == nil {
		t.Errorf("expected error")
	}
}

// Tests Set() and Get()
func TestGetForValidValue(t *testing.T) {
	kv := setup(t)
	want := []byte("yes")
	kv.Set([]byte("1234"), []byte("yes"))
	got, _ := kv.Get([]byte("1234"))
	if bytes.Compare(got, want) != 0 {
		t.Errorf("got %s, want %s", got, want)
	}
}

// TestSetReader
func TestSetReader(t *testing.T) {
	tests := []struct {
		testname  string
		key       []byte
		want      string
		expectErr bool
	}{
		{"set valid value", []byte("am key"), "hello I am a very powerful test *flex*", false},
		{"set empty key", []byte(""), "", true},
	}

	for _, tc := range tests {
		kv := setup(t)
		kv.SetReader(tc.key, strings.NewReader(tc.want))
		got, err := kv.Get(tc.key)
		if tc.expectErr && err == nil {
			t.Errorf("case: %s expected an error but did not get one", tc.testname)
		} else {
			if !tc.expectErr && err != nil {
				t.Errorf("case: %s unexpected error %v", tc.testname, err)
			}
			if bytes.Compare(got, []byte(tc.want)) != 0 {
				t.Errorf("case: %s got %s, want %s", tc.testname, got, tc.want)

			}
		}
	}
}
