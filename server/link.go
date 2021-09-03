package server

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/gliderlabs/ssh"
)

// SSHLinker implments proto.LinkTransport for the Charm SSH server.
type SSHLinker struct {
	server  *SSHServer
	account *charm.User
	session ssh.Session
}

// TokenCreated implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) TokenCreated(token charm.Token) {
	log.Println("TokenCreated")
	_ = sl.server.sendJSON(sl.session, charm.Link{
		Host:   sl.server.config.Host,
		Port:   sl.server.config.SSHPort,
		Token:  token,
		Status: charm.LinkStatusTokenCreated,
	})
}

// TokenSent implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) TokenSent(l *charm.Link) {
	log.Println("Token sent")
}

// Requested implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) Requested(l *charm.Link) (bool, error) {
	log.Println("Requested")
	_ = sl.server.sendJSON(sl.session, l)
	var msg charm.Message
	err := json.NewDecoder(sl.session).Decode(&msg)
	if err != nil {
		return false, err
	}
	log.Printf("MSG: %s", msg.Message)
	if strings.ToLower(msg.Message) == "yes" {
		return true, nil
	}
	return false, nil
}

// LinkedSameUser implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) LinkedSameUser(l *charm.Link) {
	log.Println("LinkedSameUser")
	_ = sl.server.sendJSON(sl.session, l)
}

// LinkedDifferentUser implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) LinkedDifferentUser(l *charm.Link) {
	log.Println("LinkedDifferentUser")
	_ = sl.server.sendJSON(sl.session, l)
}

// Success implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) Success(l *charm.Link) {
	log.Println("Success")
	_ = sl.server.sendJSON(sl.session, l)
}

// TimedOut implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) TimedOut(l *charm.Link) {
	log.Println("TimedOut")
	_ = sl.server.sendJSON(sl.session, l)
}

// Error implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) Error(l *charm.Link) {
	log.Println("Error")
	_ = sl.server.sendJSON(sl.session, l)
}

// RequestStart implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) RequestStart(l *charm.Link) {
	log.Println("RequestStart")
	_ = sl.server.sendJSON(sl.session, l)
}

// RequestDenied implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) RequestDenied(l *charm.Link) {
	log.Println("RequestDenied")
	_ = sl.server.sendJSON(sl.session, l)
}

// RequestValidToken implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) RequestValidToken(l *charm.Link) {
	log.Println("RequestValidToken")
	_ = sl.server.sendJSON(sl.session, l)
}

// RequestInvalidToken implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) RequestInvalidToken(l *charm.Link) {
	log.Println("RequestInvalidToken")
	_ = sl.server.sendJSON(sl.session, l)
}

// User implements the proto.LinkTransport interface for the SSHLinker.
func (sl *SSHLinker) User() *charm.User {
	log.Println("User")
	return sl.account
}

// DeleteLinkRequest implements the proto.LinkTransport interface for the SSHLinker.
func (me *SSHServer) DeleteLinkRequest(tok charm.Token) {
	delete(me.linkRequests, tok)
}

// LinkGen implements the proto.LinkTransport interface for the SSHLinker.
func (me *SSHServer) LinkGen(lt charm.LinkTransport) error {
	u := lt.User()
	tok := me.NewToken()
	linkRequest, ok := me.linkRequests[tok]
	defer me.DeleteLinkRequest(tok)
	if !ok {
		log.Printf("Making new link for token: %s\n", tok)
		linkRequest = make(chan *charm.Link)
		me.linkRequests[tok] = linkRequest
	}
	log.Printf("Token created %s", tok)
	lt.TokenCreated(tok)
	select {
	case l := <-linkRequest:
		log.Printf("Link request received %v", l)
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
			log.Printf("Link %s timed out", tok)
			lt.TimedOut(&charm.Link{Status: charm.LinkStatusTimedOut})
			return nil
		}
		if err != nil {
			return err
		}
		if approved {
			if u.CharmID == "" {
				// Create account for the link generator public key if it doesn't exist
				log.Printf("Creating account for token: %s", tok)
				u, err = me.db.UserForKey(u.PublicKey.Key, true)
				if err != nil {
					log.Printf("Create account error: %s", err)
					l.Status = charm.LinkStatusError
					me.sendLink(lt, linkRequest, l)
					return err
				}
			}
			log.Printf("Found account %s\n", u.CharmID)
			// Look up account for the link requester public key
			lu, err := me.db.UserForKey(l.RequestPubKey, false)
			if err != nil && err != charm.ErrMissingUser {
				log.Printf("Storage key lookup error: %s", err)
				l.Status = charm.LinkStatusError
				me.sendLink(lt, linkRequest, l)
				return err
			}
			if err == charm.ErrMissingUser {
				// Add the link requester's key to the link generator's account if one does not exist
				log.Printf("Link account key to account %s", u.CharmID)
				err = me.db.LinkUserKey(u, l.RequestPubKey)
				if err != nil {
					l.Status = charm.LinkStatusError
					me.sendLink(lt, linkRequest, l)
					return err
				}
				l.Status = charm.LinkStatusSuccess
			} else if lu.ID == u.ID {
				// Maybe they're already linked
				log.Printf("Key is already linked to account %s", u.CharmID)
				l.Status = charm.LinkStatusSameUser
				lt.LinkedSameUser(l)
			} else {
				// Link requester's key is linked to another acccount, merge
				log.Printf("Key is already linked to different account %s", lu.CharmID)
				err = me.db.MergeUsers(u.ID, lu.ID)
				if err != nil {
					l.Status = charm.LinkStatusError
					me.sendLink(lt, linkRequest, l)
					return err
				}
				l.Status = charm.LinkStatusSuccess
			}
			if l.Status == charm.LinkStatusSuccess {
				log.Printf("Link %s approved", tok)
				lt.Success(l)
			}
		} else {
			log.Printf("Link %s not approved", tok)
			l.Status = charm.LinkStatusRequestDenied
		}
		me.sendLink(lt, linkRequest, l)
	case <-time.After(charm.LinkTimeout):
		log.Printf("Link %s timed out", tok)
		lt.TimedOut(&charm.Link{Status: charm.LinkStatusTimedOut})
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
	}
	l.Token = charm.Token(token)
	lt.RequestStart(l)
	linkRequest, ok := me.linkRequests[l.Token]
	if ok {
		l.Status = charm.LinkStatusValidTokenRequest
		lt.RequestValidToken(l)
	} else {
		l.Status = charm.LinkStatusInvalidTokenRequest
		lt.RequestInvalidToken(l)
		return fmt.Errorf("Invalid token '%s'", token)
	}
	select {
	case linkRequest <- l:
		select {
		case lr := <-linkRequest:
			switch lr.Status {
			case charm.LinkStatusSuccess:
				lt.Success(l)
			case charm.LinkStatusSameUser:
				lt.LinkedSameUser(l)
			case charm.LinkStatusRequestDenied:
				lt.RequestDenied(l)
			default:
				log.Printf("Link error: %d", lr.Status)
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
	t, err := me.tokenBucket.NewToken(4)
	if err != nil {
		panic(err)
	}
	return charm.Token(t)
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
	log.Printf("API link gen user %s\n", u.CharmID)
	linker := &SSHLinker{
		account: u,
		session: s,
		server:  me,
	}
	err = me.LinkGen(linker)
	if err != nil {
		log.Printf("Error linking account: %s", err)
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
	log.Println("API link request")
	linker := &SSHLinker{
		session: s,
		server:  me,
	}
	ip := s.RemoteAddr().String()
	t := strings.ToUpper(s.Command()[1])
	err = me.LinkRequest(linker, key, t, ip)
	if err != nil {
		log.Printf("Error linking account: %s", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error linking account: %s", err))
		return
	}
	me.config.Stats.APILinkRequest()
}

func (me *SSHServer) handleAPILink(s ssh.Session) {
	args := s.Command()[1:]
	if len(args) == 0 {
		me.handleLinkGenAPI(s)
	} else {
		me.handleLinkRequestAPI(s)
	}
}

func (me *SSHServer) handleAPIUnlink(s ssh.Session) {
	key, err := keyText(s)
	if err != nil {
		log.Println(err)
		_ = me.sendAPIMessage(s, "Missing key")
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		log.Printf("Error fetching user: %s", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error fetching user: %s", err))
		return
	}
	log.Printf("API unlink user %s\n", u.CharmID)

	var ur charm.UnlinkRequest
	err = json.NewDecoder(s).Decode(&ur)
	if err != nil {
		log.Printf("Error unlinking account: %s", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error unlinking account: %s", err))
		return
	}
	if ur.Key == "" {
		log.Println("Error unlinking account: blank key")
		_ = me.sendAPIMessage(s, "missing key")
		return
	}
	err = me.db.UnlinkUserKey(u, ur.Key)
	if err != nil {
		log.Printf("Error unlinking account: %s", err)
		_ = me.sendAPIMessage(s, fmt.Sprintf("Error unlinking account: %s", err))
		return
	}
	me.config.Stats.APIUnlink()
}

func (me *SSHServer) sendLink(lt charm.LinkTransport, lc chan *charm.Link, l *charm.Link) {
	go func() {
		select {
		case lc <- l:
		case <-time.After(charm.LinkTimeout):
			l.Status = charm.LinkStatusTimedOut
			lt.TimedOut(l)
		}
	}()
}
