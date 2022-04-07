package kv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/server"
	"github.com/charmbracelet/keygen"
	badger "github.com/dgraph-io/badger/v3"
)

var (
	cfg *server.Config
	td  string
)

// Helpers

func startServer(t *testing.T, testName string, testFunc func(*KV, string)) {
	cfg, td = getTestServerConfig(t)
	s, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("new server error: %s", err)
	}
	go s.Start()
	kv, pn := setupKV(t)
	t.Run("health-ping", func(t *testing.T) {
		_, err := fetchURL(fmt.Sprintf("http://localhost:%d", cfg.HealthPort), 3)
		if err != nil {
			t.Fatalf("could not ping server: %s", err)
		}
	})
	t.Run(testName, func(t *testing.T) {
		testFunc(kv, pn)
	})
	t.Cleanup(func() {
		os.RemoveAll(pn)
		err := s.Close()
		if err != nil {
			log.Printf("error closing server: %s", err)
		}
	})
}

func getTestServerConfig(t *testing.T) (*server.Config, string) {
	// set up template server configurations
	cfg := server.DefaultConfig()
	td := t.TempDir()
	cfg.DataDir = filepath.Join(td, ".data")
	sp := filepath.Join(td, ".ssh")
	kp, err := keygen.NewWithWrite(sp, "charm_server", []byte(""), keygen.Ed25519)
	if err != nil {
		t.Fatalf("keygen error: %s", err)
	}
	cfg = cfg.WithKeys(kp.PublicKey, kp.PrivateKeyPEM)
	return cfg, td
}

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

func setupKV(t *testing.T) (*KV, string) {
	t.Helper()
	pn, err := ioutil.TempDir("", "charmkv")
	if err != nil {
		t.Fatal(err)
	}
	opt := badger.DefaultOptions(pn).WithLoggingLevel(badger.ERROR)
	cc := setupTestClient(t)
	kv, err := Open(cc, "test", opt)
	if err != nil {
		log.Fatal(err)
	}
	return kv, pn
}

func setupTestClient(t *testing.T) *client.Client {
	ccfg, err := client.ConfigFromEnv()
	if err != nil {
		t.Fatalf("client config from env error: %s", err)
	}
	ccfg.Host = cfg.Host
	ccfg.SSHPort = cfg.SSHPort
	ccfg.HTTPPort = cfg.HTTPPort
	ccfg.DataDir = filepath.Join(td, ".client-data")
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
	startServer(t, "get for empty DB", func(*KV, string) {
		kv, pn := setupKV(t)
		defer os.RemoveAll(pn)
		_, err := kv.Get([]byte("1234"))
		if err == nil {
			t.Errorf("expected error")
		}
	})
}

func TestGet(t *testing.T) {
	startServer(t, "get for non-empty DB", func(*KV, string) {
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
			kv, pn := setupKV(t)
			defer os.RemoveAll(pn)
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
		}
	})
}

// TestSetReader

func TestSetReader(t *testing.T) {
	startServer(t, "set reader", func(*KV, string) {
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
			kv, pn := setupKV(t)
			defer os.RemoveAll(pn)
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
		}
	})
}

// TestDelete

func TestDelete(t *testing.T) {
	startServer(t, "set reader", func(*KV, string) {
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
			kv, pn := setupKV(t)
			defer os.RemoveAll(pn)
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
		}
	})
}

// TestSync

func TestSync(t *testing.T) {
	startServer(t, "set reader", func(*KV, string) {
		kv, pn := setupKV(t)
		defer os.RemoveAll(pn)
		err := kv.Sync()
		if err != nil {
			t.Errorf("unexpected error")
		}
	})
}

// TestOptionsWithEncryption

func TestOptionsWithEncryption(t *testing.T) {
	startServer(t, "set reader", func(*KV, string) {
		_, err := OptionsWithEncryption(badger.DefaultOptions(""), []byte("1234"), -2)
		if err == nil {
			t.Errorf("expected an error")
		}
	})
}

// TestKeys

func TestKeys(t *testing.T) {
	startServer(t, "test keys", func(*KV, string) {
		tests := []struct {
			testname string
			keys     [][]byte
		}{
			{"single value", [][]byte{[]byte("one")}},
			{"two values", [][]byte{[]byte("one"), []byte("two")}},
			{"multiple values", [][]byte{[]byte("one"), []byte("two"), []byte("three")}},
		}

		for _, tc := range tests {
			kv, pn := setupKV(t)
			kv.addKeys(tc.keys)
			got, err := kv.Keys()
			if err != nil {
				t.Errorf("unexpected error")
			}
			if !compareKeyLists(got, tc.keys) {
				t.Errorf("got: %s want: %s", showKeys(got), showKeys(tc.keys))
			}
			os.RemoveAll(pn)
		}
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
		msg += "\n"
	}
	return msg
}
