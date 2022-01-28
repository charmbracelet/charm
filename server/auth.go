package server

import (
	"fmt"
	"log"
	"runtime/debug"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/wish"
	"github.com/gliderlabs/ssh"
)

func (me *SSHServer) sshMiddleware() wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			func() {
				// Recover from panics
				defer func() {
					if r := recover(); r != nil {
						me.errorLog.Printf("ssh: panic %v\n%s", r, string(debug.Stack()))
					}
				}()
				cmd := s.Command()
				if len(cmd) >= 1 {
					r := cmd[0]
					log.Printf("ssh %s\n", r)
					switch r {
					case "api-auth":
						me.handleAPIAuth(s)
					case "api-keys":
						me.handleAPIKeys(s)
					case "api-link":
						me.handleAPILink(s)
					case "api-unlink":
						me.handleAPIUnlink(s)
					case "id":
						me.handleID(s)
					case "jwt":
						me.handleJWT(s)
					}
				}
			}()
			sh(s)
		}
	}
}

func (me *SSHServer) handleAPIAuth(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		me.errorLog.Println(err)
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		me.errorLog.Println(err)
		return
	}
	log.Printf("JWT for user %s\n", u.CharmID)
	j, err := me.newJWT(u.CharmID, "charm")
	if err != nil {
		me.errorLog.Printf("Error making JWT: %s\n", err)
		return
	}

	eks, err := me.db.EncryptKeysForPublicKey(u.PublicKey)
	if err != nil {
		me.errorLog.Printf("Error fetching encrypt keys: %s\n", err)
		return
	}
	_ = me.sendJSON(s, charm.Auth{
		JWT:         j,
		ID:          u.CharmID,
		HTTPScheme:  me.config.httpScheme,
		PublicKey:   u.PublicKey.Key,
		EncryptKeys: eks,
	})
	me.config.Stats.APIAuth()
}

func (me *SSHServer) handleAPIKeys(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		me.errorLog.Println(err)
		_ = me.sendAPIMessage(s, "Missing key")
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		me.errorLog.Println(err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("API keys error: %s", err))
		return
	}
	log.Printf("API keys for user %s\n", u.CharmID)
	keys, err := me.db.KeysForUser(u)
	if err != nil {
		me.errorLog.Println(err)
		_ = me.sendAPIMessage(s, "There was a problem fetching your keys")
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

	_ = me.sendJSON(s, charm.Keys{
		ActiveKey: activeKey,
		Keys:      keys,
	})
	me.config.Stats.APIKeys()
}

func (me *SSHServer) handleID(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		me.errorLog.Println(err)
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		me.errorLog.Println(err)
		return
	}
	log.Printf("ID for user %s\n", u.CharmID)
	_, _ = s.Write([]byte(u.CharmID))
	me.config.Stats.ID()
}
