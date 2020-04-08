package charm

type LinkStatus int

const (
	LinkStatusInit LinkStatus = iota
	LinkStatusTokenCreated
	LinkStatusTokenSent
	LinkStatusRequested
	LinkStatusRequestDenied
	LinkStatusSameAccount
	LinkStatusDifferentAccount
	LinkStatusSuccess
	LinkStatusTimedOut
	LinkStatusError
	LinkStatusValidTokenRequest
	LinkStatusInvalidTokenRequest
)

type Link struct {
	Token         string     `json:"token"`
	RequestPubKey string     `json:"request_pub_key"`
	RequestAddr   string     `json:"request_addr"`
	Host          string     `json:"host"`
	Port          int        `json:"port"`
	Status        LinkStatus `json:"status"`
}

type LinkerMessage struct {
	Message string `json:"message"`
}

// LinkHandler handles linking operations
type LinkHandler interface {
	TokenCreated(*Link)
	TokenSent(*Link)
	ValidToken(*Link)
	InvalidToken(*Link)
	Request(*Link) bool
	RequestDenied(*Link)
	SameAccount(*Link)
	Success(*Link)
	Timeout(*Link)
	Error(*Link)
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
	case LinkStatusSameAccount:
		lh.SameAccount(l)
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
