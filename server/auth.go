package server

import (
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/charm"
	"github.com/dgrijalva/jwt-go"
)

type Auth struct {
	JWT         string              `json:"jwt"`
	ID          string              `json:"charm_id"`
	PublicKey   string              `json:"public_key,omitempty"`
	EncryptKeys []*charm.EncryptKey `json:"encrypt_keys,omitempty"`
}

func (me *SSHServer) HandleAPIAuth(s Session) {
	key, err := s.KeyText()
	if err != nil {
		log.Println(err)
		return
	}
	u, err := me.storage.UserForKey(key, true)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("JWT for user %s\n", u.CharmID)
	j, err := me.newJWT(u.CharmID)
	if err != nil {
		log.Printf("Error making JWT: %s\n", err)
		return
	}

	eks, err := me.storage.EncryptKeysForPublicKey(u.PublicKey)
	if err != nil {
		log.Printf("Error fetching encrypt keys: %s\n", err)
		return
	}
	_ = me.SendJSON(s, Auth{
		JWT:         j,
		ID:          u.CharmID,
		PublicKey:   u.PublicKey.Key,
		EncryptKeys: eks,
	})
	// me.config.Stats.APIAuthCalls.Inc()
}

func (me *SSHServer) HandleAPIKeys(s Session) {
	key, err := s.KeyText()
	if err != nil {
		log.Println(err)
		_ = me.SendAPIMessage(s, "Missing key")
		return
	}
	u, err := me.storage.UserForKey(key, true)
	if err != nil {
		log.Println(err)
		_ = me.SendAPIMessage(s, fmt.Sprintf("API keys error: %s", err))
		return
	}
	log.Printf("API keys for user %s\n", u.CharmID)
	keys, err := me.storage.KeysForUser(u)
	if err != nil {
		log.Println(err)
		_ = me.SendAPIMessage(s, "There was a problem fetching your keys")
		return
	}

	// Find index of the key currently in use
	activeKey := -1
	for i, k := range keys {
		if k.Key == u.PublicKey.Key {
			activeKey = i
			break
		}
	}

	_ = me.SendJSON(s, struct {
		ActiveKey int                `json:"active_key"`
		Keys      []*charm.PublicKey `json:"keys"`
	}{
		ActiveKey: activeKey,
		Keys:      keys,
	})
	// me.config.Stats.APIKeysCalls.Inc()
}

func (me *SSHServer) HandleID(s Session) {
	key, err := s.KeyText()
	if err != nil {
		log.Println(err)
		return
	}
	u, err := me.storage.UserForKey(key, true)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("ID for user %s\n", u.CharmID)
	_, _ = s.Write([]byte(u.CharmID))
	// me.config.Stats.IDCalls.Inc()
}

func (me *SSHServer) HandleJWT(s Session) {
	key, err := s.KeyText()
	if err != nil {
		log.Println(err)
		return
	}
	u, err := me.storage.UserForKey(key, true)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("JWT for user %s\n", u.CharmID)
	j, err := me.newJWT(u.CharmID)
	if err != nil {
		log.Println(err)
		return
	}
	_, _ = s.Write([]byte(j))
	// me.config.Stats.JWTCalls.Inc()
}

func (me *SSHServer) newJWT(charmID string) (string, error) {
	claims := &jwt.StandardClaims{
		Subject:   charmID,
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS512, claims).SignedString(me.jwtPrivateKey)
}
