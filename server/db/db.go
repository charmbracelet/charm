package db

import (
	"time"

	charm "github.com/charmbracelet/charm/proto"
)

// DB specifies the business logic methods a datastore must implement as the
// Charm Cloud backend.
type DB interface {
	UserForKey(key string, create bool) (*charm.User, error)
	LinkUserKey(user *charm.User, key string) error
	UnlinkUserKey(user *charm.User, key string) error
	KeysForUser(user *charm.User) ([]*charm.PublicKey, error)
	MergeUsers(userID1 int, userID2 int) error
	EncryptKeysForPublicKey(pk *charm.PublicKey) ([]*charm.EncryptKey, error)
	AddEncryptKeyForPublicKey(user *charm.User, publicKey string, globalID string, encryptedKey string, createdAt *time.Time) error
	GetUserWithID(charmID string) (*charm.User, error)
	GetUserWithName(name string) (*charm.User, error)
	SetUserName(charmID string, name string) (*charm.User, error)
	UserCount() (int, error)
	UserNameCount() (int, error)
	NextSeq(user *charm.User, name string) (uint64, error)
	GetSeq(user *charm.User, name string) (uint64, error)
	PostNews(subject string, body string, tags []string) error
	GetNews(id string) (*charm.News, error)
	GetNewsList(tag string, page int) ([]*charm.News, error)
	Close() error
}
