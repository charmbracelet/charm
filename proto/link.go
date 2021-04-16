package proto

import "time"

// LinkStatus represents a state in the linking process.
type LinkStatus int

// LinkStatus values.
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

// LinkTimeout is the length of time a Token is valid for.
const LinkTimeout = time.Minute

// Token represent the confirmation code generated during linking.
type Token string

// Link is the struct used to communicate state during the account linking
// process.
type Link struct {
	Token         Token      `json:"token"`
	RequestPubKey string     `json:"request_pub_key"`
	RequestAddr   string     `json:"request_addr"`
	Host          string     `json:"host"`
	Port          int        `json:"port"`
	Status        LinkStatus `json:"status"`
}

// LinkHandler handles linking operations for the key to be linked.
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

// LinkTransport handles linking operations for the link generation.
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
	User() *User
}

// UnlinkRequest is the message for unlinking an account from a key.
type UnlinkRequest struct {
	Key string `json:"key"`
}
