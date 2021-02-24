package server

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/charm"
)

const linkTimeout = time.Minute

type Token string

type Link struct {
	Token         Token            `json:"token"`
	RequestPubKey string           `json:"request_pub_key"`
	RequestAddr   string           `json:"request_addr"`
	Host          string           `json:"host"`
	Port          int              `json:"port"`
	Status        charm.LinkStatus `json:"status"`
}

type LinkTransport interface {
	TokenCreated(Token)
	TokenSent(*Link)
	Requested(*Link) (bool, error)
	LinkedSameUser(*Link)
	LinkedDifferentUser(*Link)
	Success(*Link)
	TimedOut(*Link)
	Error(*Link)
	RequestStart(*Link)
	RequestDenied(*Link)
	RequestInvalidToken(*Link)
	RequestValidToken(*Link)
	User() *charm.User
}

type SSHLinker struct {
	server  *SSHServer
	account *charm.User
	session Session
}

type LinkerMessage struct {
	Message string `json:"message"`
}

type UnlinkRequest struct {
	Key string `json:"key"`
}

func (sl *SSHLinker) TokenCreated(token Token) {
	log.Println("TokenCreated")
	_ = sl.server.SendJSON(sl.session, Link{
		Host:   sl.server.config.Host,
		Port:   sl.server.config.SSHPort,
		Token:  token,
		Status: charm.LinkStatusTokenCreated,
	})
}

func (sl *SSHLinker) TokenSent(l *Link) {
	log.Println("Token sent")
}

func (sl *SSHLinker) Requested(l *Link) (bool, error) {
	log.Println("Requested")
	_ = sl.server.SendJSON(sl.session, l)
	var msg LinkerMessage
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

func (sl *SSHLinker) LinkedSameUser(l *Link) {
	log.Println("LinkedSameUser")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) LinkedDifferentUser(l *Link) {
	log.Println("LinkedDifferentUser")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) Success(l *Link) {
	log.Println("Success")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) TimedOut(l *Link) {
	log.Println("TimedOut")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) Error(l *Link) {
	log.Println("Error")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) RequestStart(l *Link) {
	log.Println("RequestStart")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) RequestDenied(l *Link) {
	log.Println("RequestDenied")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) RequestValidToken(l *Link) {
	log.Println("RequestValidToken")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) RequestInvalidToken(l *Link) {
	log.Println("RequestInvalidToken")
	_ = sl.server.SendJSON(sl.session, l)
}

func (sl *SSHLinker) User() *charm.User {
	log.Println("User")
	return sl.account
}

func (me *SSHServer) DeleteLinkRequest(tok Token) {
	delete(me.linkRequests, tok)
}

func (me *SSHServer) LinkGen(lt LinkTransport) error {
	u := lt.User()
	tok := me.NewToken()
	linkRequest, ok := me.linkRequests[tok]
	defer me.DeleteLinkRequest(tok)
	if !ok {
		log.Printf("Making new link for token: %s\n", tok)
		linkRequest = make(chan *Link)
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
		case <-time.After(linkTimeout):
			log.Printf("Link %s timed out", tok)
			lt.TimedOut(&Link{Status: charm.LinkStatusTimedOut})
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
	case <-time.After(linkTimeout):
		log.Printf("Link %s timed out", tok)
		lt.TimedOut(&Link{Status: charm.LinkStatusTimedOut})
	}
	return nil
}

func (me *SSHServer) LinkRequest(lt LinkTransport, key string, token string, ip string) error {
	l := &Link{
		Host:          me.config.Host,
		RequestAddr:   ip,
		RequestPubKey: key,
		Status:        charm.LinkStatusTokenSent,
	}
	l.Token = Token(token)
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
		case <-time.After(linkTimeout):
			l.Status = charm.LinkStatusTimedOut
			lt.TimedOut(l)
		}
	case <-time.After(linkTimeout):
		l.Status = charm.LinkStatusTimedOut
		lt.TimedOut(l)
	}
	return nil
}

func (me *SSHServer) HandleLinkGenAPI(s Session) {
	key, err := s.KeyText()
	if err != nil {
		_ = me.SendAPIMessage(s, fmt.Sprintf("Missing public key %s", err))
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		_ = me.SendAPIMessage(s, fmt.Sprintf("Storage key lookup error: %s", err))
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
		_ = me.SendAPIMessage(s, fmt.Sprintf("Error linking account: %s", err))
		return
	}
	me.config.Stats.APILinkGenCalls.Inc()
}

func (me *SSHServer) HandleLinkRequestAPI(s Session) {
	key, err := s.KeyText()
	if err != nil {
		_ = me.SendAPIMessage(s, fmt.Sprintf("Missing public key %s", err))
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
		_ = me.SendAPIMessage(s, fmt.Sprintf("Error linking account: %s", err))
		return
	}
	me.config.Stats.APILinkRequestCalls.Inc()
}

func (me *SSHServer) HandleAPILink(s Session) {
	args := s.Command()[1:]
	if len(args) == 0 {
		me.HandleLinkGenAPI(s)
	} else {
		me.HandleLinkRequestAPI(s)
	}
}

func (me *SSHServer) HandleAPIUnlink(s Session) {
	key, err := s.KeyText()
	if err != nil {
		log.Println(err)
		_ = me.SendAPIMessage(s, "Missing key")
		return
	}
	u, err := me.db.UserForKey(key, true)
	if err != nil {
		log.Printf("Error fetching user: %s", err)
		_ = me.SendAPIMessage(s, fmt.Sprintf("Error fetching user: %s", err))
		return
	}
	log.Printf("API unlink user %s\n", u.CharmID)

	var ur UnlinkRequest
	err = json.NewDecoder(s).Decode(&ur)
	if err != nil {
		log.Printf("Error unlinking account: %s", err)
		_ = me.SendAPIMessage(s, fmt.Sprintf("Error unlinking account: %s", err))
		return
	}
	if ur.Key == "" {
		log.Println("Error unlinking account: blank key")
		_ = me.SendAPIMessage(s, "missing key")
		return
	}
	err = me.db.UnlinkUserKey(u, ur.Key)
	if err != nil {
		log.Printf("Error unlinking account: %s", err)
		_ = me.SendAPIMessage(s, fmt.Sprintf("Error unlinking account: %s", err))
		return
	}
	me.config.Stats.APIUnlinkCalls.Inc()
}

func (me *SSHServer) NewToken() Token {
	t, err := me.tokenBucket.NewToken(4)
	if err != nil {
		panic(err)
	}
	return Token(t)
}

func (me *SSHServer) sendLink(lt LinkTransport, lc chan *Link, l *Link) {
	go func() {
		select {
		case lc <- l:
		case <-time.After(linkTimeout):
			l.Status = charm.LinkStatusTimedOut
			lt.TimedOut(l)
		}
	}()
}
