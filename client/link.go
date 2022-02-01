package client

import (
	"encoding/json"
	"fmt"
	"os"

	charm "github.com/charmbracelet/charm/proto"
)

// LinkGen initiates a linking session.
func (cc *Client) LinkGen(lh charm.LinkHandler) error {
	s, err := cc.sshSession()
	if err != nil {
		return err
	}
	defer s.Close()
	out, err := s.StdoutPipe()
	if err != nil {
		return err
	}
	in, err := s.StdinPipe()
	if err != nil {
		return err
	}
	err = s.Start("api-link")
	if err != nil {
		return err
	}

	// initialize link request on server
	var lr charm.Link
	dec := json.NewDecoder(out)
	err = dec.Decode(&lr)
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr) {
		return nil
	}

	// waiting for link request, do we want to approve it?
	err = dec.Decode(&lr)
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr) {
		return nil
	}

	// send approval response
	var lm charm.Message
	enc := json.NewEncoder(in)
	if lh.Request(&lr) {
		lm = charm.Message{Message: "yes"}
	} else {
		lm = charm.Message{Message: "no"}
	}
	err = enc.Encode(lm)
	if err != nil {
		return err
	}
	if lm.Message == "no" {
		return nil
	}

	// get server response
	err = dec.Decode(&lr)
	if err != nil {
		return err
	}
	err = cc.SyncEncryptKeys()
	if err != nil {
		return err
	}
	checkLinkStatus(lh, &lr)
	return nil
}

// Link joins in on a linking session initiated by LinkGen.
func (cc *Client) Link(lh charm.LinkHandler, code string) error {
	s, err := cc.sshSession()
	if err != nil {
		return err
	}
	defer s.Close()
	out, err := s.StdoutPipe()
	if err != nil {
		return err
	}
	err = s.Start(fmt.Sprintf("api-link %s", code))
	if err != nil {
		return err
	}
	var lr charm.Link
	dec := json.NewDecoder(out)
	err = dec.Decode(&lr) // Start Request
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr) {
		return nil
	}

	err = dec.Decode(&lr) // Token Check
	if err != nil {
		return err
	}
	if !checkLinkStatus(lh, &lr) {
		return nil
	}

	err = dec.Decode(&lr) // Results
	if err != nil {
		return err
	}
	err = cc.SyncEncryptKeys()
	if err != nil {
		return err
	}
	checkLinkStatus(lh, &lr)
	return nil
}

// SyncEncryptKeys re-encodes all of the encrypt keys associated for this
// public key with all other linked public keys.
func (cc *Client) SyncEncryptKeys() error {
	cc.InvalidateAuth()
	eks, err := cc.EncryptKeys()
	if err != nil {
		return err
	}
	cks, err := cc.AuthorizedKeysWithMetadata()
	if err != nil {
		return err
	}
	for _, k := range cks.Keys {
		for _, ek := range eks {
			err := cc.addEncryptKey(k.Key, ek.ID, ek.Key, ek.CreatedAt)
			if err != nil {
				return err
			}
		}
	}
	return cc.deleteUserData()
}

// TODO find a better place for this, or do something more sophisticated than
// just wiping it out.
func (cc *Client) deleteUserData() error {
	dd, err := DataPath(cc.Config.Host)
	if err != nil {
		return err
	}
	// TODO add any other directories that need wiping
	kvd := fmt.Sprintf("%s/kv", dd)
	return os.RemoveAll(kvd)
}

func checkLinkStatus(lh charm.LinkHandler, l *charm.Link) bool {
	switch l.Status {
	case charm.LinkStatusTokenCreated:
		lh.TokenCreated(l)
	case charm.LinkStatusTokenSent:
		lh.TokenSent(l)
	case charm.LinkStatusValidTokenRequest:
		lh.ValidToken(l)
	case charm.LinkStatusInvalidTokenRequest:
		lh.InvalidToken(l)
		return false
	case charm.LinkStatusRequestDenied:
		lh.RequestDenied(l)
		return false
	case charm.LinkStatusSameUser:
		lh.SameUser(l)
	case charm.LinkStatusSuccess:
		lh.Success(l)
	case charm.LinkStatusTimedOut:
		lh.Timeout(l)
		return false
	case charm.LinkStatusError:
		lh.Error(l)
		return false
	}
	return true
}
