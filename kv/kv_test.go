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

func TestMain(m *testing.M) {
	var err error
	cfg, td, err = getTestServerConfig()
	if err != nil {
		log.Fatalf("new server error: %s", err)
	}

	s, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("new server error: %s", err)
	}

	go s.Start()

	_, pn, err := setupKV()
	if err != nil {
		log.Fatalln("setup kv error", err)
	}

	if _, err = fetchURL(fmt.Sprintf("http://localhost:%d", cfg.HealthPort), 3); err != nil {
		log.Fatalln("ping error", err)
	}

	code := m.Run()
	os.RemoveAll(pn)
	if err := s.Close(); err != nil {
		log.Printf("error closing server: %s", err)
	}
	os.Exit(code)
}

// Helpers

func getTestServerConfig() (*server.Config, string, error) {
	// set up template server configurations
	cfg := server.DefaultConfig()
	td, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, "", err
	}
	cfg.DataDir = filepath.Join(td, ".data")
	sp := filepath.Join(td, ".ssh")
	kp, err := keygen.NewWithWrite(sp, "charm_server", []byte(""), keygen.Ed25519)
	if err != nil {
		return nil, "", err
	}
	cfg = cfg.WithKeys(kp.PublicKey, kp.PrivateKeyPEM)
	return cfg, td, nil
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

func setupKV() (*KV, string, error) {
	pn, err := ioutil.TempDir("", "charmkv")
	if err != nil {
		return nil, "", err
	}
	opt := badger.DefaultOptions(pn).WithLoggingLevel(badger.ERROR)
	cc, err := setupTestClient()
	if err != nil {
		return nil, "", err
	}
	kv, err := Open(cc, "test", opt)
	return kv, pn, err
}

func setupTestClient() (*client.Client, error) {
	ccfg, err := client.ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	ccfg.Host = cfg.Host
	ccfg.SSHPort = cfg.SSHPort
	ccfg.HTTPPort = cfg.HTTPPort
	ccfg.DataDir = filepath.Join(td, ".client-data")
	return client.NewClient(ccfg)
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
	kv, pn, err := setupKV()
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(pn)
	if _, err := kv.Get([]byte("1234")); err == nil {
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
		kv, pn, err := setupKV()
		if err != nil {
			t.Error(err)
		}
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
		kv, pn, err := setupKV()
		if err != nil {
			t.Error(err)
		}
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
}

//
// // TestDelete
//
// func TestDelete(t *testing.T) {
// 	startServer(t, "set reader", func(*KV, string) {
// 		tests := []struct {
// 			testname  string
// 			key       []byte
// 			value     []byte
// 			expectErr bool
// 		}{
// 			{"valid key", []byte("hello"), []byte("value"), false},
// 			{"empty key with value", []byte{}, []byte("value"), true},
// 			{"empty key no value", []byte{}, []byte{}, true},
// 		}
//
// 		for _, tc := range tests {
// 			kv, pn := setupKV(t)
// 			defer os.RemoveAll(pn)
// 			kv.Set(tc.key, tc.value)
// 			if tc.expectErr {
// 				if err := kv.Delete(tc.key); err == nil {
// 					t.Errorf("%s: expected error", tc.testname)
// 				}
// 			} else {
// 				if err := kv.Delete(tc.key); err != nil {
// 					t.Errorf("%s: unexpected error in Delete %v", tc.testname, err)
// 				}
// 				want := []byte{} // want an empty result
// 				if get, _ := kv.Get(tc.key); bytes.Compare(get, want) != 0 {
// 					t.Errorf("%s: expected an empty string %s, got %s", tc.testname, want, get)
// 				}
// 			}
// 		}
// 	})
// }
//
// // TestSync
//
// func TestSync(t *testing.T) {
// 	startServer(t, "set reader", func(*KV, string) {
// 		kv, pn := setupKV(t)
// 		defer os.RemoveAll(pn)
// 		err := kv.Sync()
// 		if err != nil {
// 			t.Errorf("unexpected error")
// 		}
// 	})
// }
//
// // TestOptionsWithEncryption
//
// func TestOptionsWithEncryption(t *testing.T) {
// 	startServer(t, "set reader", func(*KV, string) {
// 		_, err := OptionsWithEncryption(badger.DefaultOptions(""), []byte("1234"), -2)
// 		if err == nil {
// 			t.Errorf("expected an error")
// 		}
// 	})
// }
//
// // TestKeys
//
// func TestKeys(t *testing.T) {
// 	startServer(t, "test keys", func(*KV, string) {
// 		tests := []struct {
// 			testname string
// 			keys     [][]byte
// 		}{
// 			{"single value", [][]byte{[]byte("one")}},
// 			{"two values", [][]byte{[]byte("one"), []byte("two")}},
// 			{"multiple values", [][]byte{[]byte("one"), []byte("two"), []byte("three")}},
// 		}
//
// 		for _, tc := range tests {
// 			kv, pn := setupKV(t)
// 			kv.addKeys(tc.keys)
// 			got, err := kv.Keys()
// 			if err != nil {
// 				t.Errorf("unexpected error")
// 			}
// 			if !compareKeyLists(got, tc.keys) {
// 				t.Errorf("got: %s want: %s", showKeys(got), showKeys(tc.keys))
// 			}
// 			os.RemoveAll(pn)
// 		}
// 	})
// }
//
// func (kv *KV) addKeys(values [][]byte) {
// 	for _, val := range values {
// 		kv.Set(val, []byte("hello"))
// 	}
// }
//
// func compareKeyLists(a, b [][]byte) bool {
// 	if len(a) != len(b) {
// 		return false
// 	}
// 	for i := range a {
// 		if bytes.Compare(a[i], b[i]) != 0 {
// 			return false
// 		}
// 	}
// 	return true
// }
//
// func showKeys(keys [][]byte) string {
// 	msg := ""
// 	for _, key := range keys {
// 		msg += string(key)
// 		msg += "\n"
// 	}
// 	return msg
// }
