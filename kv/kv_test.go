package kv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/keygen"
	badger "github.com/dgraph-io/badger/v3"
)

var (
	clientTD string
	cfg      *server.Config
)

// Helpers

func TestMain(m *testing.M) {
	var clientTD string
	var s *server.Server
	cfg = getTestServerConfig()
	s, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("new server error: %s", err)
	}
	go s.Start()
	_, clientTD = setupKV(setupTestClient(cfg))
	code := m.Run()
	os.RemoveAll(clientTD)
	if err = s.Close(); err != nil {
		log.Printf("error closing server: %s", err)
	}
	os.Exit(code)
}

// healthPing check that the server has started
func healthPing(cfg *server.Config) {
	_, err := fetchURL(fmt.Sprintf("http://localhost:%d", cfg.HealthPort), 3)
	if err != nil {
		log.Fatalf("could not ping server: %s", err)
	}
}

// getTestServerConfig create the test server
func getTestServerConfig() *server.Config {
	cfg := server.DefaultConfig()
	td, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatalf("unable to init temp dir: %s", err)
	}
	cfg.DataDir = filepath.Join(td, ".data")
	sp := filepath.Join(td, ".ssh")
	kp, err := keygen.NewWithWrite(sp, "charm_server", []byte(""), keygen.Ed25519)
	if err != nil {
		log.Fatalf("keygen error: %s", err)
	}
	cfg = cfg.WithKeys(kp.PublicKey, kp.PrivateKeyPEM)
	return cfg
}

// fetchURL check if we can successfully do a GET request to test server
func fetchURL(url string, retries int) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil {
		if retries > 0 {
			time.Sleep(time.Second)
			return fetchURL(url, retries-1)
		}
		return nil, err
	}
	if resp.StatusCode != 200 {
		return resp, fmt.Errorf("bad http status code: %d", resp.StatusCode)
	}
	return resp, nil
}

func setupKV(cc *client.Client) (*KV, string) {
	var err error
	clientTD, err = ioutil.TempDir("", "charmkv")
	if err != nil {
		log.Fatal(err)
	}
	opt := badger.DefaultOptions(clientTD).WithLoggingLevel(badger.ERROR)
	kv, err := Open(cc, "test", opt)
	if err != nil {
		log.Fatal(err)
	}
	return kv, clientTD
}

func setupTestClient(cfg *server.Config) *client.Client {
	ccfg, err := client.ConfigFromEnv()
	if err != nil {
		log.Fatalf("client config from env error: %s", err)
	}
	ccfg.Host = cfg.Host
	ccfg.SSHPort = cfg.SSHPort
	ccfg.HTTPPort = cfg.HTTPPort
	ccfg.DataDir = filepath.Join(clientTD, ".client-data")
	cl, err := client.NewClient(ccfg)
	if err != nil {
		log.Printf("unable to create new client: %s", err)
	}
	return cl
}

// TestOpenWithDefaults

func TestOpenWithDefaults(t *testing.T) {
	tests := []string{
		"one",
		"",
	}
	for _, tc := range tests {
		_, err := OpenWithDefaults(tc)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

// TestGet

func TestGetForEmptyDB(t *testing.T) {
	kv, _ := setupKV(setupTestClient(cfg))
	_, err := kv.Get([]byte("1234"))
	if err == nil {
		t.Errorf("expected error")
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		testname  string
		key       []byte
		want      []byte
		expectErr bool
	}{
		{"valid kv pair", []byte("1234"), []byte("valid"), false},
		{"invalid key", []byte{}, []byte{}, true},
	}

	for _, tc := range tests {
		kv, clientTD := setupKV(setupTestClient(cfg))
		kv.Set(tc.key, tc.want)
		got, err := kv.Get(tc.key)
		if tc.expectErr {
			if err == nil {
				t.Errorf("%s: expected error", tc.testname)
			}
		} else {
			if err != nil {
				t.Errorf("%s: unexpected error %v", tc.testname, err)
			}
			if bytes.Compare(got, tc.want) != 0 {
				t.Errorf("%s: got %s, want %s", tc.testname, got, tc.want)
			}
		}
		os.RemoveAll(clientTD)
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
		kv, clientTD := setupKV(setupTestClient(cfg))
		kv.SetReader(tc.key, strings.NewReader(tc.want))
		got, err := kv.Get(tc.key)
		if tc.expectErr {
			if err == nil {
				t.Errorf("case: %s expected an error but did not get one", tc.testname)
			}
		} else {
			if err != nil {
				t.Errorf("case: %s unexpected error %v", tc.testname, err)
			}
			if bytes.Compare(got, []byte(tc.want)) != 0 {
				t.Errorf("case: %s got %s, want %s", tc.testname, got, tc.want)
			}
		}
		os.RemoveAll(clientTD)
	}
}

// TestDelete

func TestDelete(t *testing.T) {
	tests := []struct {
		testname  string
		key       []byte
		value     []byte
		expectErr bool
	}{
		{"valid key", []byte("hello"), []byte("value"), false},
		{"empty key with value", []byte{}, []byte("value"), true},
		{"empty key no value", []byte{}, []byte{}, true},
	}

	for _, tc := range tests {
		kv, clientTD := setupKV(setupTestClient(cfg))
		kv.Set(tc.key, tc.value)
		if tc.expectErr {
			if err := kv.Delete(tc.key); err == nil {
				t.Errorf("%s: expected error", tc.testname)
			}
		} else {
			if err := kv.Delete(tc.key); err != nil {
				t.Errorf("%s: unexpected error in Delete %v", tc.testname, err)
			}
			want := []byte{} // want an empty result
			if get, _ := kv.Get(tc.key); bytes.Compare(get, want) != 0 {
				t.Errorf("%s: expected an empty string %s, got %s", tc.testname, want, get)
			}
		}
		os.RemoveAll(clientTD)
	}
}

// TestSync

func TestSync(t *testing.T) {
	kv, _ := setupKV(setupTestClient(cfg))
	err := kv.Sync()
	if err != nil {
		t.Errorf("unexpected error")
	}
}

// TestOptionsWithEncryption

func TestOptionsWithEncryption(t *testing.T) {
	_, err := OptionsWithEncryption(badger.DefaultOptions(""), []byte("1234"), -2)
	if err == nil {
		t.Errorf("expected an error")
	}
}

// TestKeys

func TestKeys(t *testing.T) {
	tests := []struct {
		testname string
		keys     [][]byte
	}{
		{"single value", [][]byte{[]byte("one")}},
		{"two values", [][]byte{[]byte("one"), []byte("two")}},
	}
	for _, tc := range tests {
		kv, clientTD := setupKV(setupTestClient(cfg))
		kv.addKeys(tc.keys)
		got, err := kv.Keys()
		if err != nil {
			t.Errorf("unexpected error")
		}
		if !compareKeyLists(got, tc.keys) {
			t.Errorf("got: %s want: %s", showKeys(got), showKeys(tc.keys))
		}
		os.RemoveAll(clientTD)
	}
}

func TestMultipleKeys(t *testing.T) {
	kv, _ := setupKV(setupTestClient(cfg))
	keys := [][]byte{[]byte("one"), []byte("two"), []byte("three")}
	kv.addKeys(keys)
	got, err := kv.Keys()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	sortKeys(got)
	sortKeys(keys)
	if !compareKeyLists(got, keys) {
		t.Errorf("got: %s want: %s", showKeys(got), showKeys(keys))
	}
}

func sortKeys(ss [][]byte) {
	sort.Slice(ss, func(i, j int) bool {
		return bytes.Compare(ss[i], ss[j]) < 0
	})
}

func (kv *KV) addKeys(values [][]byte) {
	for _, val := range values {
		kv.Set(val, []byte("hello"))
	}
}

func compareKeyLists(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	// TODO: make order not matter
	for i := range a {
		if bytes.Compare(a[i], b[i]) != 0 {
			return false
		}
	}
	return true
}

func showKeys(keys [][]byte) string {
	msg := ""
	for _, key := range keys {
		msg += string(key)
		msg += ", "
	}
	return msg
}
