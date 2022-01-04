package client

import (
	"encoding/json"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/dgrijalva/jwt-go"
)

// Auth will authenticate a client and cache the result. It will return a
// proto.Auth with the JWT and encryption keys for a user.
func (cc *Client) Auth() (*charm.Auth, error) {
	cc.authLock.Lock()
	defer cc.authLock.Unlock()

	cfg := cc.Config
	if cc.claims == nil || cc.claims.Valid() != nil {
		auth := &charm.Auth{}
		s, err := cc.sshSession()
		if err != nil {
			return nil, charm.ErrAuthFailed{Err: err}
		}
		defer s.Close()

		b, err := s.Output("api-auth")
		if err != nil {
			return nil, charm.ErrAuthFailed{Err: err}
		}
		err = json.Unmarshal(b, auth)
		if err != nil {
			return nil, charm.ErrAuthFailed{Err: err}
		}
		// Set HTTP scheme from the server if it's not set.
		if cfg.HTTPScheme == "" {
			cfg.HTTPScheme = auth.HTTPScheme
		}
		p := &jwt.Parser{}
		token, _, err := p.ParseUnverified(auth.JWT, &jwt.StandardClaims{})
		if err != nil {
			return nil, charm.ErrAuthFailed{Err: err}
		}
		cc.claims = token.Claims.(*jwt.StandardClaims)
		cc.auth = auth
		if err != nil {
			return nil, charm.ErrAuthFailed{Err: err}
		}
	}
	return cc.auth, nil
}

// InvalidateAuth clears the JWT auth cache, forcing subsequent Auth() to fetch
// a new JWT from the server.
func (cc *Client) InvalidateAuth() {
	cc.authLock.Lock()
	defer cc.authLock.Unlock()
	cc.claims = nil
	cc.auth = nil
}
