package charm

import (
	"encoding/json"
	"fmt"
)

// LinkStatus represents a state in the linking process.
type LinkStatus int

const (
	LinkStatusInit LinkStatus = iota
	LinkStatusTokenCreated
	LinkStatusTokenSent
	LinkStatusRequested
	LinkStatusRequestDenied
	LinkStatusSameUser
	LinkStatusDifferentUser
	LinkStatusSuccess
	LinkStatusTimedOut
	LinkStatusError
	LinkStatusValidTokenRequest
	LinkStatusInvalidTokenRequest
)

// Link is the struct used to communicate state during the account linking
// process.
type Link struct {
	Token         string     `json:"token"`
	RequestPubKey string     `json:"request_pub_key"`
	RequestAddr   string     `json:"request_addr"`
	Host          string     `json:"host"`
	Port          int        `json:"port"`
	Status        LinkStatus `json:"status"`
}

// LinkerMessage is used for communicating errors and data in the linking
// process.
type LinkerMessage struct {
	Message string `json:"message"`
}

// LinkHandler handles linking operations.
type LinkHandler interface {
	TokenCreated(*Link)
	TokenSent(*Link)
	ValidToken(*Link)
	InvalidToken(*Link)
	Request(*Link) bool
	RequestDenied(*Link)
	SameUser(*Link)
	Success(*Link)
	Timeout(*Link)
	Error(*Link)
}

// LinkGen initiates a linking session.
func (cc *Client) LinkGen(lh LinkHandler) error {
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
	var lr Link
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
	var lm LinkerMessage
	enc := json.NewEncoder(in)
	if lh.Request(&lr) {
		lm = LinkerMessage{"yes"}
	} else {
		lm = LinkerMessage{"no"}
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
	if !checkLinkStatus(lh, &lr) {
		return nil
	}
	return cc.SyncEncryptKeys()
}

// Link joins in on a linking session initiated by LinkGen.
func (cc *Client) Link(lh LinkHandler, code string) error {
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
	var lr Link
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
// public key with all other linked publick keys.
func (cc *Client) SyncEncryptKeys() error {
	cc.InvalidateAuth()
	eks, err := cc.encryptKeys()
	if err != nil {
		return err
	}
	cks, err := cc.AuthorizedKeysWithMetadata()
	if err != nil {
		return err
	}
	for _, k := range cks.Keys {
		for _, ek := range eks {
			err := cc.addEncryptKey(k.Key, ek.GlobalID, ek.Key)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func checkLinkStatus(lh LinkHandler, l *Link) bool {
	switch l.Status {
	case LinkStatusTokenCreated:
		lh.TokenCreated(l)
	case LinkStatusTokenSent:
		lh.TokenSent(l)
	case LinkStatusValidTokenRequest:
		lh.ValidToken(l)
	case LinkStatusInvalidTokenRequest:
		lh.InvalidToken(l)
		return false
	case LinkStatusRequestDenied:
		lh.RequestDenied(l)
		return false
	case LinkStatusSameUser:
		lh.SameUser(l)
	case LinkStatusSuccess:
		lh.Success(l)
	case LinkStatusTimedOut:
		lh.Timeout(l)
		return false
	case LinkStatusError:
		lh.Error(l)
		return false
	}
	return true
}
