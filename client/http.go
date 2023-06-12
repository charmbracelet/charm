package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	charm "github.com/charmbracelet/charm/proto"
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
		"Content-Type": []string{"application/json"},
	}
	resp, err := cc.AuthedRequest(method, path, headers, buf)
	if err != nil {
		return err
	}
	if respBody != nil {
		defer resp.Body.Close() // nolint:errcheck
		dec := json.NewDecoder(resp.Body)
		return dec.Decode(respBody)
	}
	return nil
}

// AuthedRequest sends an authorized request to the Charm and Glow HTTP servers.
func (cc *Client) AuthedRequest(method string, path string, headers http.Header, reqBody io.Reader) (*http.Response, error) {
	client := &http.Client{}
	cfg := cc.Config
	auth, err := cc.Auth()
	if err != nil {
		return nil, err
	}
	jwt := auth.JWT
	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%d%s", cc.httpScheme, cfg.Host, cfg.HTTPPort, path), reqBody)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		for _, vv := range v {
			req.Header.Add(k, vv)
			if k == "Content-Length" {
				req.ContentLength, _ = strconv.ParseInt(vv, 10, 64)
			}
		}
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if statusCode := resp.StatusCode; statusCode >= 300 {
		err = fmt.Errorf("server error: %d %s", statusCode, http.StatusText(statusCode))
		// try to decode the error message
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
			msg := charm.Message{}
			_ = json.NewDecoder(resp.Body).Decode(&msg)
			if msg.Message != "" {
				err = fmt.Errorf("%s: %s", err, msg.Message)
			}
		}
		return resp, err
	}
	return resp, nil
}

// AuthedRawRequest sends an authorized request with no request body to the Charm and Glow HTTP servers.
func (cc *Client) AuthedRawRequest(method string, path string) (*http.Response, error) {
	return cc.AuthedRequest(method, path, nil, nil)
}
