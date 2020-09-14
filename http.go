package charm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// AuthedRequest sends an authorized request to the Charm and Glow HTTP servers.
func (cc *Client) AuthedRequest(method string, host string, port int, path string, reqBody interface{}, respBody interface{}) error {
	client := &http.Client{}
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s:%d%s", host, port, path), buf)
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
		return ErrNameTaken
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
