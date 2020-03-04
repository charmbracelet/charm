package charm

type LinkStatus int

const (
	LinkStatusInit LinkStatus = iota
	LinkStatusTokenCreated
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
