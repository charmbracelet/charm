package client

import (
	"fmt"
	"net/url"
	"strings"

	charm "github.com/charmbracelet/charm/proto"
)

func (cc *Client) NewsList(tags []string, page int) ([]*charm.News, error) {
	var nl []*charm.News

	if tags == nil {
		tags = []string{"server"}
	}
	tq := url.QueryEscape(strings.Join(tags, ","))
	err := cc.AuthedJSONRequest("GET", fmt.Sprintf("/v1/news?page=%d&tags=%s", page, tq), nil, &nl)
	if err != nil {
		return nil, err
	}
	return nl, nil
}

func (cc *Client) News(id string) (*charm.News, error) {
	var n *charm.News
	err := cc.AuthedJSONRequest("GET", fmt.Sprintf("/v1/news/%s", url.QueryEscape(id)), nil, &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
