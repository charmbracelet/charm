package charm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrorPageOutOfBounds = errors.New("page must be a value of 1 or greater")

// MarkdownsByCreatedAtDesc sorts markdown documents by date in descending
// order. It implements sort.Interface for []Markdown based on the CreatedAt
// field.
type MarkdownsByCreatedAtDesc []*Markdown

// Sort implementation for MarkdownByCreatedAt
func (m MarkdownsByCreatedAtDesc) Len() int           { return len(m) }
func (m MarkdownsByCreatedAtDesc) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m MarkdownsByCreatedAtDesc) Less(i, j int) bool { return m[i].CreatedAt.After(*m[j].CreatedAt) }

type Markdown struct {
	ID           int        `json:"id"`
	EncryptKeyID string     `json:"encrypt_key_id"`
	Note         string     `json:"note"`
	Body         string     `json:"body,omitempty"`
	CreatedAt    *time.Time `json:"created_at"`
}

func (cc *Client) GetNews(page int) ([]*Markdown, error) {
	if page < 1 {
		return nil, ErrorPageOutOfBounds
	}
	var news []*Markdown
	err := cc.makeAPIRequest("GET", fmt.Sprintf("news?page=%d", page), nil, &news)
	if err != nil {
		return nil, err
	}
	return news, nil
}

func (cc *Client) GetNewsMarkdown(markdownID int) (*Markdown, error) {
	var md Markdown
	err := cc.makeAPIRequest("GET", fmt.Sprintf("news/%d", markdownID), nil, &md)
	if err != nil {
		return nil, err
	}
	return &md, nil
}

func (cc *Client) GetStash(page int) ([]*Markdown, error) {
	if page < 1 {
		return nil, ErrorPageOutOfBounds
	}
	var stash []*Markdown
	auth, err := cc.Auth()
	if err != nil {
		return nil, err
	}
	err = cc.makeAPIRequest("GET", fmt.Sprintf("%s/stash?page=%d", auth.CharmID, page), nil, &stash)
	if err != nil {
		return nil, err
	}
	for i, md := range stash {
		en, err := cc.Decrypt(md.EncryptKeyID, []byte(md.Note))
		if err != nil {
			return nil, err
		}
		stash[i].Note = string(en)
	}
	return stash, nil
}

func (cc *Client) GetStashMarkdown(markdownID int) (*Markdown, error) {
	var md Markdown
	auth, err := cc.Auth()
	if err != nil {
		return nil, err
	}
	err = cc.makeAPIRequest("GET", fmt.Sprintf("%s/stash/%d", auth.CharmID, markdownID), nil, &md)
	if err != nil {
		return nil, err
	}
	eb, err := cc.Decrypt(md.EncryptKeyID, []byte(md.Body))
	if err != nil {
		return nil, err
	}
	md.Body = string(eb)
	en, err := cc.Decrypt(md.EncryptKeyID, []byte(md.Note))
	if err != nil {
		return nil, err
	}
	md.Note = string(en)
	return &md, nil
}

func (cc *Client) StashMarkdown(note string, body string) error {
	auth, err := cc.Auth()
	if err != nil {
		return err
	}
	eb, gid, err := cc.Encrypt([]byte(body))
	en, gid, err := cc.Encrypt([]byte(note))
	md := &Markdown{Note: string(en), Body: string(eb), EncryptKeyID: gid}
	return cc.makeAPIRequest("POST", fmt.Sprintf("%s/stash", auth.CharmID), md, nil)
}

func (cc *Client) DeleteMarkdown(markdownID int) error {
	auth, err := cc.Auth()
	if err != nil {
		return err
	}
	return cc.makeAPIRequest("DELETE", fmt.Sprintf("%s/stash/%d", auth.CharmID, markdownID), nil, nil)
}

func (cc *Client) SetMarkdownNote(markdownID int, note string) error {
	auth, err := cc.Auth()
	if err != nil {
		return err
	}
	md, err := cc.GetStashMarkdown(markdownID)
	if err != nil {
		return err
	}
	en, _, err := cc.EncryptWithKey(md.EncryptKeyID, []byte(note))
	md.Note = string(en)
	return cc.makeAPIRequest("PUT", fmt.Sprintf("%s/stash/%d", auth.CharmID, markdownID), md, nil)
}

func (cc *Client) authorizeRequest(req *http.Request) error {
	auth, err := cc.Auth()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("bearer %s", auth.JWT))
	return nil
}

func (cc *Client) makeAPIRequest(method string, apiPath string, body interface{}, result interface{}) error {
	var buf *bytes.Buffer
	var err error
	var req *http.Request
	client := &http.Client{}
	url := fmt.Sprintf("%s:%d/v1/%s", cc.config.GlowHost, cc.config.GlowPort, apiPath)
	if body != nil {
		buf = &bytes.Buffer{}
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return err
		}
		req, err = http.NewRequest(method, url, buf)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http server error %d", resp.StatusCode)
	}
	if result != nil {
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(result)
		if err != nil {
			return err
		}
	}
	return nil
}
