package stats

import "context"

// Stats provides an interface that different stats backend can implement to
// track server usage.
type Stats interface {
	Start() error
	Shutdown(context.Context) error
	APILinkGen()
	APILinkRequest()
	APIUnlink()
	APIAuth()
	APIKeys()
	LinkGen()
	LinkRequest()
	Keys()
	ID()
	JWT()
	GetUserByID()
	GetUser()
	SetUserName()
	GetNewsList()
	GetNews()
	PostNews()
	Close() error
}
