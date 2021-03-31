package proto

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
