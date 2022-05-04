package stats

// Stats provides an interface that different stats backend can implement to
// track server usage.
type Stats interface {
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
	FSFileRead(id string, size int64)
	FSFileWritten(id string, size int64)
	Close() error
}
