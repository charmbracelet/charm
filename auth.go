package charm

import (
	"encoding/json"

	"github.com/dgrijalva/jwt-go"
)

// Auth is the authenticated user's charm id and jwt returned from the ssh server.
type Auth struct {
	CharmID     string        `json:"charm_id"`
	JWT         string        `json:"jwt"`
	PublicKey   string        `json:"public_key"`
	EncryptKeys []*EncryptKey `json:"encrypt_keys"`
	claims      *jwt.StandardClaims
}

// Auth returns the Auth struct for a client session. It will renew and cache
// the Charm ID JWT.
func (cc *Client) Auth() (*Auth, error) {
	cc.authLock.Lock()
	defer cc.authLock.Unlock()

	if cc.auth.claims == nil || cc.auth.claims.Valid() != nil {
		auth := &Auth{}
		s, err := cc.sshSession()
		if err != nil {
			return nil, ErrAuthFailed{err}
		}
		defer s.Close()

		b, err := s.Output("api-auth")
		if err != nil {
			return nil, ErrAuthFailed{err}
		}
		err = json.Unmarshal(b, auth)
		if err != nil {
			return nil, ErrAuthFailed{err}
		}

		token, err := jwt.ParseWithClaims(auth.JWT, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
			return cc.jwtPublicKey, nil
		})
		if err != nil {
			return nil, ErrAuthFailed{err}
		}

		auth.claims = token.Claims.(*jwt.StandardClaims)
		cc.auth = auth
		if err != nil {
			return nil, ErrAuthFailed{err}
		}
	}
	return cc.auth, nil
}

// InvalidateAuth clears the JWT auth cache, forcing subsequent Auth() to fetch
// a new JWT from the server.
func (cc *Client) InvalidateAuth() {
	cc.authLock.Lock()
	defer cc.authLock.Unlock()
	cc.auth.claims = nil
}
