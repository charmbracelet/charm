package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charmbracelet/charm"
)

// AuthedRequest sends an authorized request to the Charm and Glow HTTP servers.
func (cc *Client) AuthedRequest(method string, path string, reqBody interface{}, respBody interface{}) error {
	client := &http.Client{}
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return err
	}
	cfg := cc.Config
	req, err := http.NewRequest(method, fmt.Sprintf("%s://%s:%d%s", cfg.HTTPScheme, cfg.Host, cfg.HTTPPort, path), buf)
	if err != nil {
		return err
	}
	jwt, err := cc.JWT()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", jwt))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusConflict {
		return charm.ErrNameTaken
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	if respBody != nil {
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		return dec.Decode(respBody)
	}
	return nil
}
