package charm

import (
	"bytes"
	"encoding/base64"
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
func (m MarkdownsByCreatedAtDesc) Less(i, j int) bool { return m[i].CreatedAt.After(m[j].CreatedAt) }

type Markdown struct {
	ID           int       `json:"id"`
	EncryptKeyID string    `json:"encrypt_key_id"`
	Note         string    `json:"note"`
	Body         string    `json:"body,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
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
		dm, err := cc.decryptMarkdown(md)
		if err != nil {
			return nil, err
		}
		stash[i] = dm

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
	mdDec, err := cc.decryptMarkdown(&md)
	if err != nil {
		return nil, err
	}
	return mdDec, nil
}

func (cc *Client) StashMarkdown(note string, body string) error {
	auth, err := cc.Auth()
	if err != nil {
		return err
	}

	md := &Markdown{Note: note, Body: body}
	md, err = cc.encryptMarkdown(md)
	if err != nil {
		return err
	}

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
	md.Note = note
	md, err = cc.encryptMarkdown(md)
	if err != nil {
		return err
	}

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

func (cc *Client) encryptMarkdown(md *Markdown) (*Markdown, error) {
	mde := &Markdown{}
	mde.ID = md.ID
	mde.CreatedAt = md.CreatedAt
	eb, gid, err := cc.Encrypt([]byte(md.Body))
	if err != nil {
		return nil, err
	}
	encBody := base64.StdEncoding.EncodeToString(eb)
	mde.Body = encBody
	mde.EncryptKeyID = gid
	if md.Note != "" {
		ed, _, err := cc.EncryptWithKey(gid, []byte(md.Note))
		if err != nil {
			return nil, err
		}
		encNote := base64.StdEncoding.EncodeToString(ed)
		mde.Note = encNote
	}
	return mde, nil
}

func (cc *Client) decryptMarkdown(mde *Markdown) (*Markdown, error) {
	md := &Markdown{}
	md.ID = mde.ID
	md.CreatedAt = mde.CreatedAt
	if mde.EncryptKeyID == "" {
		md.Note = mde.Note
		md.Body = mde.Body
		return md, nil
	}
	if mde.Note != "" {
		encNote, err := base64.StdEncoding.DecodeString(mde.Note)
		if err != nil {
			return nil, err
		}
		decNote, err := cc.Decrypt(mde.EncryptKeyID, encNote)
		if err != nil {
			return nil, err
		}
		md.Note = string(decNote)
	}
	if mde.Body != "" {
		encBody, err := base64.StdEncoding.DecodeString(mde.Body)
		if err != nil {
			return nil, err
		}
		decBody, err := cc.Decrypt(mde.EncryptKeyID, encBody)
		if err != nil {
			return nil, err
		}
		md.Body = string(decBody)
	}
	return md, nil
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
