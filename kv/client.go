package kv

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/charm/client"
)

func encryptKeyFromCharmClient(cc *client.Client) ([]byte, error) {
	var auth *client.Auth
	var err error
	auth, err = cc.Auth()
	if err != nil {
		return nil, err
	}
	if len(auth.EncryptKeys) == 0 {
		err = cc.GenerateEncryptKeys()
		if err != nil {
			return nil, err
		}
		auth, err = cc.Auth()
		if err != nil {
			return nil, err
		}
	}
	ek := []byte(auth.EncryptKeys[0].Key)
	if len(ek) < 32 {
		return nil, fmt.Errorf("Encryption key is too short")
	}
	return []byte(ek)[0:32], nil
}

func (kv *KV) seqStorageKey(seq uint64) string {
	return strings.Join([]string{kv.name, fmt.Sprintf("%d", seq)}, "/")
}

func (kv *KV) encryptAndBackupSeq(from uint64, at uint64) error {
	buf := bytes.NewBuffer(nil)
	ew, err := kv.cc.NewEncryptedWriter("", buf)
	s := kv.DB.NewStreamAt(math.MaxUint64)
	_, err = s.Backup(ew, from)
	if err != nil {
		return err
	}
	ew.Close()
	return kv.storeDatalog(kv.seqStorageKey(at), buf)
}

func (kv *KV) decryptAndRestoreSeq(seq uint64) error {
	// there is never a zero seq
	if seq == 0 {
		return nil
	}
	r, err := kv.getDatalog(kv.seqStorageKey(seq))
	defer r.Close()
	if err != nil {
		return err
	}
	dr, err := kv.cc.NewDecryptedReader("", r)
	if err != nil {
		return err
	}
	// TODO DB.Load() should be called on a database that is not running any
	// other concurrent transactions while it is running.
	return kv.DB.Load(dr, 1)
}

func (kv *KV) getSeq(name string) (uint64, error) {
	var sm *charm.SeqMsg
	err := kv.cc.AuthedRequest("GET", fmt.Sprintf("/v1/seq/%s", name), nil, &sm)
	if err != nil {
		return 0, err
	}
	return sm.Seq, nil
}

func (kv *KV) nextSeq(name string) (uint64, error) {
	var sm *charm.SeqMsg
	err := kv.cc.AuthedRequest("POST", fmt.Sprintf("/v1/seq/%s", name), nil, &sm)
	if err != nil {
		return 0, err
	}
	return sm.Seq, nil
}

func (kv *KV) storeDatalog(key string, r io.Reader) error {
	buf := bytes.NewBuffer(nil)
	w := multipart.NewWriter(buf)
	fw, err := w.CreateFormFile("data", key)
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, r)
	if err != nil {
		return err
	}
	w.Close()
	cfg := kv.cc.Config
	path := fmt.Sprintf("%s://%s:%d/v1/datalog/%s", cfg.HTTPScheme, cfg.Host, cfg.HTTPPort, key)
	req, err := http.NewRequest("POST", path, buf)
	if err != nil {
		return err
	}
	jwt, err := kv.cc.JWT()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}

func (kv *KV) getDatalog(key string) (io.ReadCloser, error) {
	cfg := kv.cc.Config
	path := fmt.Sprintf("%s://%s:%d/v1/datalog/%s", cfg.HTTPScheme, cfg.Host, cfg.HTTPPort, key)
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	jwt, err := kv.cc.JWT()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
}
