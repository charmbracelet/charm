package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/charmbracelet/ssh"
	"github.com/muesli/toktok"
)

// SSHLinker implments proto.LinkTransport for the Charm SSH server.
type SSHLinker struct {
	server  *SSHServer
	account *charm.User
	session ssh.Session
}

// TokenCreated implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) TokenCreated(token charm.Token) {
	log.Debug("TokenCreated")
	_ = sl.server.sendJSON(sl.session, charm.Link{
		Host:   sl.server.config.Host,
		Port:   sl.server.config.SSHPort,
		Token:  token,
		Status: charm.LinkStatusTokenCreated,
	})
}

// TokenSent implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) TokenSent(_ *charm.Link) {
	log.Debug("Token sent")
}

// Requested implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) Requested(l *charm.Link) (bool, error) {
	log.Debug("Requested")
	_ = sl.server.sendJSON(sl.session, l)
	var msg charm.Message
	err := json.NewDecoder(sl.session).Decode(&msg)
	if err != nil {
		return false, err
	}
	log.Debug("MSG", "msg", msg.Message)
	if strings.ToLower(msg.Message) == "yes" {
		return true, nil
	}
	return false, nil
}

// LinkedSameUser implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) LinkedSameUser(l *charm.Link) {
	log.Debug("LinkedSameUser")
	_ = sl.server.sendJSON(sl.session, l)
}

// LinkedDifferentUser implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) LinkedDifferentUser(l *charm.Link) {
	log.Debug("LinkedDifferentUser")
	_ = sl.server.sendJSON(sl.session, l)
}

// Success implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) Success(l *charm.Link) {
	log.Debug("Success")
	_ = sl.server.sendJSON(sl.session, l)
}

// TimedOut implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) TimedOut(l *charm.Link) {
	log.Debug("TimedOut")
	_ = sl.server.sendJSON(sl.session, l)
}

// Error implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) Error(l *charm.Link) {
	log.Debug("Error")
	_ = sl.server.sendJSON(sl.session, l)
}

// RequestStart implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) RequestStart(l *charm.Link) {
	log.Debug("RequestStart")
	_ = sl.server.sendJSON(sl.session, l)
}

// RequestDenied implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) RequestDenied(l *charm.Link) {
	log.Debug("RequestDenied")
	_ = sl.server.sendJSON(sl.session, l)
}

// RequestValidToken implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) RequestValidToken(l *charm.Link) {
	log.Debug("RequestValidToken")
	_ = sl.server.sendJSON(sl.session, l)
}

// RequestInvalidToken implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) RequestInvalidToken(l *charm.Link) {
	log.Debug("RequestInvalidToken")
	_ = sl.server.sendJSON(sl.session, l)
}

// User implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) User() *charm.User {
	log.Debug("User")
	return sl.account
}

// LinkGen implements the proto.LinkTransport interface for the SSHLinker.
func (me *SSHServer) LinkGen(lt charm.LinkTransport) error {
	u := lt.User()
	tok := me.NewToken()
	defer me.db.DeleteToken(tok) // nolint:errcheck
	me.linkQueue.InitLinkRequest(tok)
	defer me.linkQueue.DeleteLinkRequest(tok)
	linkRequest, err := me.linkQueue.WaitLinkRequest(tok)
	if err != nil {
		return err
	}
	log.Debug("Token created", "token", tok)
	lt.TokenCreated(tok)
	select {
	case l := <-linkRequest:
		log.Debug("Link request received", "link", l)
		var err error
		var approved bool
		ch := make(chan bool, 1)
		go func() {
			approved, err = lt.Requested(l)
			ch <- approved
		}()
		select {
		case <-ch:
		case <-time.After(charm.LinkTimeout):
			log.Debug("Link timed out", "token", tok)
			l.Status = charm.LinkStatusTimedOut
			lt.TimedOut(l)
			return nil
		}
		if err != nil {
			return err
		}
		if approved {
			if u.CharmID == "" {
				// Create account for the link generator public key if it doesn't exist
				log.Debug("Creating account for token", "token", tok)
				u, err = me.db.UserForKey(u.PublicKey.Key, true)
				if err != nil {
					log.Error("Create account error", "err", err)
					l.Status = charm.LinkStatusError
					me.linkQueue.SendLinkRequest(lt, linkRequest, l)
					return err
				}
			}
			log.Debug("Found account", "id", u.CharmID)
			// Look up account for the link requester public key
			lu, err := me.db.UserForKey(l.RequestPubKey, false)
			if err != nil && err != charm.ErrMissingUser {
				log.Error("Storage key lookup error", "err", err)
				l.Status = charm.LinkStatusError
				me.linkQueue.SendLinkRequest(lt, linkRequest, l)
				return err
			}
			if err == charm.ErrMissingUser {
				// Add the link requester's key to the link generator's account if one does not exist
				log.Debug("Link account key to account", "id", u.CharmID)
				err = me.db.LinkUserKey(u, l.RequestPubKey)
				if err != nil {
					l.Status = charm.LinkStatusError
					me.linkQueue.SendLinkRequest(lt, linkRequest, l)
					return err
				}
				l.Status = charm.LinkStatusSuccess
			} else if lu.ID == u.ID {
				// Maybe they're already linked
				log.Debug("Key is already linked to account", "id", u.CharmID)
				l.Status = charm.LinkStatusSameUser
				lt.LinkedSameUser(l)
			} else {
				// Link requester's key is linked to another acccount, merge
				log.Debug("Key is already linked to different account", "id", lu.CharmID)
				err = me.db.MergeUsers(u.ID, lu.ID)
				if err != nil {
					l.Status = charm.LinkStatusError
					me.linkQueue.SendLinkRequest(lt, linkRequest, l)
					return err
				}
				l.Status = charm.LinkStatusSuccess
			}
			if l.Status == charm.LinkStatusSuccess {
				log.Debug("Link approved", "token", tok)
				lt.Success(l)
			}
		} else {
			log.Debug("Link not approved", "token", tok)
			l.Status = charm.LinkStatusRequestDenied
		}
		me.linkQueue.SendLinkRequest(lt, linkRequest, l)
	case <-time.After(charm.LinkTimeout):
		log.Debug("Link timed out", "token", tok)
		lt.TimedOut(&charm.Link{Token: tok, Status: charm.LinkStatusTimedOut})
	}
	return nil
}

// LinkRequest implements the proto.LinkTransport interface for the SSHLinker.
func (me *SSHServer) LinkRequest(lt charm.LinkTransport, key string, token string, ip string) error {
	l := &charm.Link{
		Host:          me.config.Host,
		RequestAddr:   ip,
		RequestPubKey: key,
		Status:        charm.LinkStatusTokenSent,
		Token:         charm.Token(token),
	}
	lt.RequestStart(l)
	linkRequest, err := me.linkQueue.WaitLinkRequest(l.Token)
	if err != nil || !me.linkQueue.ValidateLinkRequest(l.Token) {
		l.Status = charm.LinkStatusInvalidTokenRequest
		lt.RequestInvalidToken(l)
		return fmt.Errorf("invalid token '%s'", token)
	}
	l.Status = charm.LinkStatusValidTokenRequest
	lt.RequestValidToken(l)
	select {
	case linkRequest <- l:
		select {
		case lr := <-linkRequest:
			l.Status = lr.Status
			switch lr.Status {
			case charm.LinkStatusSuccess:
				lt.Success(l)
			case charm.LinkStatusSameUser:
				lt.LinkedSameUser(l)
			case charm.LinkStatusRequestDenied:
				lt.RequestDenied(l)
			default:
				log.Error("Link error", "status", lr.Status)
				l.Status = charm.LinkStatusError
				lt.Error(l)
			}
		case <-time.After(charm.LinkTimeout):
			l.Status = charm.LinkStatusTimedOut
			lt.TimedOut(l)
		}
	case <-time.After(charm.LinkTimeout):
		l.Status = charm.LinkStatusTimedOut
		lt.TimedOut(l)
	}
	return nil
}

// NewToken creates and returns a new Token.
func (me *SSHServer) NewToken() charm.Token {
	t := toktok.GenerateToken(6, []rune("ABCDEFHJKLMNPRSTUWXY369"))
	tok := charm.Token(t)
	err := me.db.SetToken(tok)
	if err != nil && err != charm.ErrTokenExists {
		panic(err)
	}
	if err == charm.ErrTokenExists {
		return me.NewToken()
	}
	return tok
}

func (me *SSHServer) handleLinkGenAPI(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		_ = me.sendAPIMessage(s, fmt.Sprintf("Missing public key %s", err))
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		_ = me.sendAPIMessage(s, fmt.Sprintf("Storage key lookup error: %s", err))
		return
	}
	log.Debug("API link gen user", "id", u.CharmID)
	linker := &SSHLinker{
		account: u,
		session: s,
		server:  me,
	}
	err = me.LinkGen(linker)
	if err != nil {
		log.Error("Error linking account", "err", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error linking account: %s", err))
		return
	}
	me.config.Stats.APILinkGen()
}

func (me *SSHServer) handleLinkRequestAPI(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		_ = me.sendAPIMessage(s, fmt.Sprintf("Missing public key %s", err))
		return
	}
	log.Info("API link request")
	linker := &SSHLinker{
		session: s,
		server:  me,
	}
	ip := s.RemoteAddr().String()
	t := strings.ToUpper(s.Command()[1])
	err = me.LinkRequest(linker, key, t, ip)
	if err != nil {
		log.Error("Error linking account", "err", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error linking account: %s", err))
		return
	}
	me.config.Stats.APILinkRequest()
}

func (me *SSHServer) handleAPILink(s ssh.Session) {
	if args := s.Command()[1:]; len(args) == 0 {
		me.handleLinkGenAPI(s)
	} else {
		me.handleLinkRequestAPI(s)
	}
}

func (me *SSHServer) handleAPIUnlink(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		log.Info(err)
		_ = me.sendAPIMessage(s, "Missing key")
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		log.Error("Error fetching user", "err", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error fetching user: %s", err))
		return
	}
	log.Error("API unlink user", "id", u.CharmID)

	var ur charm.UnlinkRequest
	err = json.NewDecoder(s).Decode(&ur)
	if err != nil {
		log.Error("Error unlinking account", "err", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error unlinking account: %s", err))
		return
	}
	if ur.Key == "" {
		log.Error("Error unlinking account: blank key")
		_ = me.sendAPIMessage(s, "missing key")
		return
	}
	err = me.db.UnlinkUserKey(u, ur.Key)
	if err != nil {
		log.Error("Error unlinking account", "err", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error unlinking account: %s", err))
		return
	}
	me.config.Stats.APIUnlink()
}

type channelLinkQueue struct {
	s            *SSHServer
	linkRequests map[charm.Token]chan *charm.Link
}

// InitLinkRequest implements the proto.LinkQueue interface for the channelLinkQueue.
func (s *channelLinkQueue) InitLinkRequest(t charm.Token) {
	if _, ok := s.linkRequests[t]; !ok {
		log.Error("Making new link for token", "token", t)
		lr := make(chan *charm.Link)
		s.linkRequests[t] = lr
	}
}

// ValidateLinkRequest implements the proto.LinkQueue interface for the channelLinkQueue.
func (s *channelLinkQueue) ValidateLinkRequest(t charm.Token) bool {
	_, err := s.WaitLinkRequest(t)
	return err == nil
}

// WaitLinkRequest implements the proto.LinkQueue interface for the channelLinkQueue.
func (s *channelLinkQueue) WaitLinkRequest(t charm.Token) (chan *charm.Link, error) {
	lr, ok := s.linkRequests[t]
	if !ok {
		return nil, fmt.Errorf("no link request for token: %s", t)
	}
	return lr, nil
}

// SendLinkRequest implements the proto.LinkQueue interface for the channelLinkQueue.
func (s *channelLinkQueue) SendLinkRequest(lt charm.LinkTransport, lc chan *charm.Link, l *charm.Link) {
	go func() {
		select {
		case lc <- l:
		case <-time.After(charm.LinkTimeout):
			l.Status = charm.LinkStatusTimedOut
			lt.TimedOut(l)
		}
	}()
}

// DeleteLinkRequest implements the proto.LinkTransport interface for the channelLinkQueue.
func (s *channelLinkQueue) DeleteLinkRequest(tok charm.Token) {
	delete(s.linkRequests, tok)
}
