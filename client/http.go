package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrRequestTooLarge is an error for a request that is too large.
type ErrRequestTooLarge struct {
	Size  int64
	Limit int64
}

func (err ErrRequestTooLarge) Error() string {
	return fmt.Sprintf("request too large: %d > %d", err.Size, err.Limit)
}

// AuthedRequest sends an authorized JSON request to the Charm and Glow HTTP servers.
func (cc *Client) AuthedJSONRequest(method string, path string, reqBody interface{}, respBody interface{}) error {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return err
	}
	headers := http.Header{
		"Content-Type": {"application/json"},
	}
	resp, err := cc.AuthedRequest(method, path, headers, buf)
	if err != nil {
		return err
	}
	if respBody != nil {
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		return dec.Decode(respBody)
	}
	return nil
}

// AuthedRequest sends an authorized request to the Charm and Glow HTTP servers.
func (cc *Client) AuthedRequest(method string, path string, headers http.Header, reqBody io.Reader) (*http.Response, error) {
	var maxRequestSize int64
	if strings.HasPrefix(path, "/v1/fs") {
		maxRequestSize = 1 << 30
	} else {
		maxRequestSize = 1 << 20
	}
	client := &http.Client{}
	cfg := cc.Config
	auth, err := cc.Auth()
	if err != nil {
		return nil, err
	}
	jwt := auth.JWT
	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%d%s", cfg.HTTPScheme, cfg.Host, cfg.HTTPPort, path), reqBody)
	if err != nil {
		return nil, err
	}
	if req.ContentLength > maxRequestSize {
		return nil, ErrRequestTooLarge{Size: req.ContentLength, Limit: maxRequestSize}
	}
	for k, v := range headers {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("server error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp, nil
}

// AuthedRawRequest sends an authorized request with no request body to the Charm and Glow HTTP servers.
func (cc *Client) AuthedRawRequest(method string, path string) (*http.Response, error) {
	return cc.AuthedRequest(method, path, nil, nil)
}
