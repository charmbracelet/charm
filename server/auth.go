package server

import (
	"fmt"

	"github.com/charmbracelet/log"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

func (me *SSHServer) sshMiddleware() wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			cmd := s.Command()
			if len(cmd) >= 1 {
				r := cmd[0]
				log.Debug("ssh", "cmd", r)
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
			sh(s)
		}
	}
}

func (me *SSHServer) handleAPIAuth(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		me.errorLog.Print(err)
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		me.errorLog.Print(err)
		return
	}
	log.Debug("JWT for user", "id", u.CharmID)
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
	httpScheme := me.config.httpURL().Scheme
	_ = me.sendJSON(s, charm.Auth{
		JWT:         j,
		ID:          u.CharmID,
		HTTPScheme:  httpScheme,
		PublicKey:   u.PublicKey.Key,
		EncryptKeys: eks,
	})
	me.config.Stats.APIAuth()
}

func (me *SSHServer) handleAPIKeys(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		me.errorLog.Print(err)
		_ = me.sendAPIMessage(s, "Missing key")
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		me.errorLog.Print(err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("API keys error: %s", err))
		return
	}
	log.Debug("API keys for user", "id", u.CharmID)
	keys, err := me.db.KeysForUser(u)
	if err != nil {
		me.errorLog.Print(err)
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
		me.errorLog.Print(err)
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		me.errorLog.Print(err)
		return
	}
	log.Debug("ID for user", "id", u.CharmID)
	_, _ = s.Write([]byte(u.CharmID))
	me.config.Stats.ID()
}
