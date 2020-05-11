package charm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	apiPrefix = "v1"
)

type Markdown struct {
	ID        int        `json:"id"`
	Note      string     `json:"note"`
	Body      string     `json:"body,omitempty"`
	CreatedAt *time.Time `json:"created_at"`
}

func (cc *Client) authorizeRequest(req *http.Request) error {
	auth, err := cc.Auth()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", auth.JWT))
	return nil
}

func (cc *Client) GetStash() ([]*Markdown, error) {
	var stash []*Markdown
	client := &http.Client{}
	auth, err := cc.Auth()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s:%d/%s/%s/stash", cc.config.GlowHost, cc.config.GlowPort, apiPrefix, auth.CharmID), nil)
	if err != nil {
		return nil, err
	}
	if cc.authorizeRequest(req) != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server error")
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&stash)
	if err != nil {
		return nil, err
	}
	return stash, nil
}

func (cc *Client) GetMarkdown(markdownID int) (*Markdown, error) {
	var md Markdown
	client := &http.Client{}
	auth, err := cc.Auth()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s:%d/%s/%s/stash/%d", cc.config.GlowHost, cc.config.GlowPort, apiPrefix, auth.CharmID, markdownID), nil)
	if err != nil {
		return nil, err
	}
	if cc.authorizeRequest(req) != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server error")
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&md)
	if err != nil {
		return nil, err
	}
	return &md, nil
}

func (cc *Client) StashMarkdown(note string, body string) error {
	md := &Markdown{Note: note, Body: body}
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(md)
	if err != nil {
		return err
	}
	client := &http.Client{}
	auth, err := cc.Auth()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s:%d/%s/%s/stash", cc.config.GlowHost, cc.config.GlowPort, apiPrefix, auth.CharmID), buf)
	if err != nil {
		return err
	}
	err = cc.authorizeRequest(req)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("server error")
	}
	return nil
}

func (cc *Client) DeleteMarkdown(markdownID int) error {
	client := &http.Client{}
	auth, err := cc.Auth()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s:%d/%s/%s/stash/%d", cc.config.GlowHost, cc.config.GlowPort, apiPrefix, auth.CharmID, markdownID), nil)
	if err != nil {
		return err
	}
	if cc.authorizeRequest(req) != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("server error")
	}
	return nil
}
