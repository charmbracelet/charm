package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/log"

	charm "github.com/charmbracelet/charm/proto"
	"github.com/google/uuid"
	"modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

const (
	// The DB default file name.
	DbName = "charm_sqlite.db"
	// The DB default connection options.
	DbOptions = "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
)

// DB is the database struct.
type DB struct {
	db *sql.DB
}

// NewDB creates a new DB in the given path.
func NewDB(path string) *DB {
	var err error
	log.Debug("Opening SQLite db", "path", path)
	db, err := sql.Open("sqlite", path+DbOptions)
	if err != nil {
		panic(err)
	}
	d := &DB{db: db}
	err = d.CreateDB()
	if err != nil {
		panic(err)
	}
	return d
}

// UserCount returns the number of users.
func (me *DB) UserCount() (int, error) {
	var c int
	r := me.db.QueryRow(sqlCountUsers)
	if err := r.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// UserNameCount returns the number of users with a user name set.
func (me *DB) UserNameCount() (int, error) {
	var c int
	r := me.db.QueryRow(sqlCountUserNames)
	if err := r.Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// GetUserWithID returns the user for the given id.
func (me *DB) GetUserWithID(charmID string) (*charm.User, error) {
	r := me.db.QueryRow(sqlSelectUserWithCharmID, charmID)
	u, err := me.scanUser(r)
	if err == sql.ErrNoRows {
		return nil, charm.ErrMissingUser
	}
	return u, nil
}

// GetUserWithName returns the user for the given name.
func (me *DB) GetUserWithName(name string) (*charm.User, error) {
	r := me.db.QueryRow(sqlSelectUserWithName, name)
	u, err := me.scanUser(r)
	if err == sql.ErrNoRows {
		return nil, charm.ErrMissingUser
	}
	return u, nil
}

// SetUserName sets a user name for the given user id.
func (me *DB) SetUserName(charmID string, name string) (*charm.User, error) {
	var u *charm.User
	log.Debug("Setting name for user", "name", name, "id", charmID)
	err := me.WrapTransaction(func(tx *sql.Tx) error {
		// nolint: godox
		// TODO: this should be handled with unique constraints in the database instead.
		var err error
		r := me.selectUserWithName(tx, name)
		u, err = me.scanUser(r)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			r := me.selectUserWithCharmID(tx, charmID)
			u, err = me.scanUser(r)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == sql.ErrNoRows {
				return charm.ErrMissingUser
			}

			err = me.updateUser(tx, charmID, name)
			if err != nil {
				return err
			}

			r = me.selectUserWithName(tx, name)
			u, err = me.scanUser(r)
			if err != nil {
				return err
			}
		}
		if u.CharmID != charmID {
			return charm.ErrNameTaken
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// UserForKey returns the user for the given key, or optionally creates a new user with it.
func (me *DB) UserForKey(key string, create bool) (*charm.User, error) {
	pk := &charm.PublicKey{}
	u := &charm.User{}
	err := me.WrapTransaction(func(tx *sql.Tx) error {
		var err error
		r := me.selectPublicKey(tx, key)
		err = r.Scan(&pk.ID, &pk.UserID, &pk.Key)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows && !create {
			return charm.ErrMissingUser
		}
		if err == sql.ErrNoRows {
			log.Debug("Creating user for key", "key", charm.PublicKeySha(key))
			err = me.createUser(tx, key)
			if err != nil {
				return err
			}
		}
		r = me.selectPublicKey(tx, key)
		err = r.Scan(&pk.ID, &pk.UserID, &pk.Key)
		if err != nil {
			return err
		}

		r = me.selectUserWithID(tx, pk.UserID)
		u, err = me.scanUser(r)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			return charm.ErrMissingUser
		}
		u.PublicKey = pk
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// AddEncryptKeyForPublicKey adds an ecrypted key to the user.
func (me *DB) AddEncryptKeyForPublicKey(u *charm.User, pk string, gid string, ek string, ca *time.Time) error {
	log.Debug("Adding encrypted key for user", "key", gid, "time", ca, "id", u.CharmID)
	return me.WrapTransaction(func(tx *sql.Tx) error {
		u2, err := me.UserForKey(pk, false)
		if err != nil {
			return err
		}
		if u2.ID != u.ID {
			return fmt.Errorf("trying to add encrypted key for unauthorized user")
		}

		r := me.selectEncryptKey(tx, u2.PublicKey.ID, gid)
		ekr := &charm.EncryptKey{}
		err = r.Scan(&ekr.ID, &ekr.Key, &ekr.CreatedAt)
		if err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			return me.insertEncryptKey(tx, ek, gid, u2.PublicKey.ID, ca)
		}
		log.Debug("Encrypt key already exists for public key, skipping", "key", gid)
		return nil
	})
}

// EncryptKeysForPublicKey returns the encrypt keys for the given user.
func (me *DB) EncryptKeysForPublicKey(pk *charm.PublicKey) ([]*charm.EncryptKey, error) {
	var ks []*charm.EncryptKey
	err := me.WrapTransaction(func(tx *sql.Tx) error {
		rs, err := me.selectEncryptKeys(tx, pk.ID)
		if err != nil {
			return err
		}
		if rs.Err() != nil {
			return rs.Err()
		}
		defer rs.Close() // nolint:errcheck
		for rs.Next() {
			k := &charm.EncryptKey{}
			err := rs.Scan(&k.ID, &k.Key, &k.CreatedAt)
			if err != nil {
				return err
			}
			ks = append(ks, k)
		}
		return nil
	})
	if err != nil {
		return ks, err
	}
	return ks, nil
}

// LinkUserKey links a user to a key.
func (me *DB) LinkUserKey(user *charm.User, key string) error {
	ks := charm.PublicKeySha(key)
	log.Debug("Linking user and key", "id", user.CharmID, "key", ks)
	return me.WrapTransaction(func(tx *sql.Tx) error {
		return me.insertPublicKey(tx, user.ID, key)
	})
}

// UnlinkUserKey unlinks the key from the user.
func (me *DB) UnlinkUserKey(user *charm.User, key string) error {
	ks := charm.PublicKeySha(key)
	log.Debug("Unlinking user key", "id", user.CharmID, "key", ks)
	return me.WrapTransaction(func(tx *sql.Tx) error {
		err := me.deleteUserPublicKey(tx, user.ID, key)
		if err != nil {
			return err
		}
		r := me.selectNumberUserPublicKeys(tx, user.ID)
		var count int
		err = r.Scan(&count)
		if err != nil {
			return err
		}
		if count == 0 {
			log.Debug("Removing last key for account, deleting", "id", user.CharmID)
			// nolint: godox
			// TODO: Where to put glow stuff
			// err := me.deleteUserStashMarkdown(tx, user.ID)
			// if err != nil {
			// 	return err
			// }
			return me.deleteUser(tx, user.ID)
		}
		return nil
	})
}

// KeysForUser returns all user's public keys.
func (me *DB) KeysForUser(user *charm.User) ([]*charm.PublicKey, error) {
	var keys []*charm.PublicKey
	log.Debug("Getting keys for user", "id", user.CharmID)
	err := me.WrapTransaction(func(tx *sql.Tx) error {
		rs, err := me.selectUserPublicKeys(tx, user.ID)
		if err != nil {
			return err
		}
		defer rs.Close() // nolint:errcheck

		for rs.Next() {
			k := &charm.PublicKey{}
			err := rs.Scan(&k.ID, &k.Key, &k.CreatedAt)
			if err != nil {
				return err
			}

			keys = append(keys, k)
		}
		if rs.Err() != nil {
			return rs.Err()
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// GetSeq returns the named sequence.
func (me *DB) GetSeq(u *charm.User, name string) (uint64, error) {
	var seq uint64
	var err error
	err = me.WrapTransaction(func(tx *sql.Tx) error {
		seq, err = me.selectNamedSeq(tx, u.ID, name)
		if err == sql.ErrNoRows {
			seq, err = me.incNamedSeq(tx, u.ID, name)
		}
		return err
	})
	if err != nil {
		return 0, err
	}
	return seq, nil
}

// NextSeq increments the sequence and returns.
func (me *DB) NextSeq(u *charm.User, name string) (uint64, error) {
	var seq uint64
	var err error
	err = me.WrapTransaction(func(tx *sql.Tx) error {
		seq, err = me.incNamedSeq(tx, u.ID, name)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return seq, nil
}

// GetNews returns the server news.
func (me *DB) GetNews(id string) (*charm.News, error) {
	n := &charm.News{}
	i, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	err = me.WrapTransaction(func(tx *sql.Tx) error {
		r := me.selectNews(tx, i)
		return r.Scan(&n.ID, &n.Subject, &n.Body, &n.CreatedAt)
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

// GetNewsList returns the list of server news.
func (me *DB) GetNewsList(tag string, page int) ([]*charm.News, error) {
	var ns []*charm.News
	err := me.WrapTransaction(func(tx *sql.Tx) error {
		rs, err := me.selectNewsList(tx, tag, page)
		if err != nil {
			return err
		}
		if rs.Err() != nil {
			return rs.Err()
		}
		defer rs.Close() // nolint:errcheck
		for rs.Next() {
			n := &charm.News{}
			err := rs.Scan(&n.ID, &n.Subject, &n.CreatedAt)
			if err != nil {
				return err
			}
			ns = append(ns, n)
		}
		return nil
	})
	return ns, err
}

// PostNews publish news to the server.
func (me *DB) PostNews(subject string, body string, tags []string) error {
	return me.WrapTransaction(func(tx *sql.Tx) error {
		return me.insertNews(tx, subject, body, tags)
	})
}

// MergeUsers merge two users into a single one.
func (me *DB) MergeUsers(userID1 int, userID2 int) error {
	return me.WrapTransaction(func(tx *sql.Tx) error {
		err := me.updateMergePublicKeys(tx, userID1, userID2)
		if err != nil {
			return err
		}

		return me.deleteUser(tx, userID2)
	})
}

// SetToken creates the given token.
func (me *DB) SetToken(token charm.Token) error {
	return me.WrapTransaction(func(tx *sql.Tx) error {
		err := me.insertToken(tx, string(token))
		if err != nil {
			serr, ok := err.(*sqlite.Error)
			if ok && serr.Code() == sqlitelib.SQLITE_CONSTRAINT_UNIQUE {
				return charm.ErrTokenExists
			}
		}
		return err
	})
}

// DeleteToken deletes the given token.
func (me *DB) DeleteToken(token charm.Token) error {
	return me.WrapTransaction(func(tx *sql.Tx) error {
		return me.deleteToken(tx, string(token))
	})
}

// CreateDB creates the database.
func (me *DB) CreateDB() error {
	return me.WrapTransaction(func(tx *sql.Tx) error {
		err := me.createUserTable(tx)
		if err != nil {
			return err
		}
		err = me.createPublicKeyTable(tx)
		if err != nil {
			return err
		}
		err = me.createNamedSeqTable(tx)
		if err != nil {
			return err
		}
		err = me.createEncryptKeyTable(tx)
		if err != nil {
			return err
		}
		err = me.createNewsTable(tx)
		if err != nil {
			return err
		}
		err = me.createNewsTagTable(tx)
		if err != nil {
			return err
		}
		err = me.createTokenTable(tx)
		if err != nil {
			return err
		}
		return nil
	})
}

// Close the db.
func (me *DB) Close() error {
	log.Debug("Closing db")
	return me.db.Close()
}

func (me *DB) createUser(tx *sql.Tx, key string) error {
	charmID := uuid.New().String()
	err := me.insertUser(tx, charmID)
	if err != nil {
		return err
	}
	r := me.selectUserWithCharmID(tx, charmID)
	u, err := me.scanUser(r)
	if err != nil {
		return err
	}
	return me.insertPublicKey(tx, u.ID, key)
}

func (me *DB) insertUser(tx *sql.Tx, charmID string) error {
	_, err := tx.Exec(sqlInsertUser, charmID)
	return err
}

func (me *DB) insertPublicKey(tx *sql.Tx, userID int, key string) error {
	_, err := tx.Exec(sqlInsertPublicKey, userID, key)
	return err
}

func (me *DB) insertEncryptKey(tx *sql.Tx, key string, globalID string, publicKeyID int, createdAt *time.Time) error {
	var err error
	if createdAt == nil {
		_, err = tx.Exec(sqlInsertEncryptKey, key, globalID, publicKeyID)
	} else {
		_, err = tx.Exec(sqlInsertEncryptKeyWithDate, key, globalID, publicKeyID, createdAt)
	}
	return err
}

func (me *DB) insertNews(tx *sql.Tx, subject string, body string, tags []string) error {
	r, err := tx.Exec(sqlInsertNews, subject, body)
	if err != nil {
		return err
	}
	nid, err := r.LastInsertId()
	if err != nil {
		return err
	}
	for _, tag := range tags {
		_, err = tx.Exec(sqlInsertNewsTag, nid, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

func (me *DB) insertToken(tx *sql.Tx, token string) error {
	_, err := tx.Exec(sqlInsertToken, token)
	return err
}

func (me *DB) selectNamedSeq(tx *sql.Tx, userID int, name string) (uint64, error) {
	var seq uint64
	r := tx.QueryRow(sqlSelectNamedSeq, userID, name)
	if err := r.Scan(&seq); err != nil {
		return 0, err
	}
	return seq, nil
}

func (me *DB) incNamedSeq(tx *sql.Tx, userID int, name string) (uint64, error) {
	_, err := tx.Exec(sqlIncNamedSeq, userID, name)
	if err != nil {
		return 0, err
	}
	return me.selectNamedSeq(tx, userID, name)
}

func (me *DB) updateUser(tx *sql.Tx, charmID string, name string) error {
	_, err := tx.Exec(sqlUpdateUser, name, charmID)
	return err
}

func (me *DB) selectUserWithName(tx *sql.Tx, name string) *sql.Row {
	return tx.QueryRow(sqlSelectUserWithName, name)
}

func (me *DB) selectUserWithCharmID(tx *sql.Tx, charmID string) *sql.Row {
	return tx.QueryRow(sqlSelectUserWithCharmID, charmID)
}

func (me *DB) selectUserWithID(tx *sql.Tx, userID int) *sql.Row {
	return tx.QueryRow(sqlSelectUserWithID, userID)
}

func (me *DB) selectUserPublicKeys(tx *sql.Tx, userID int) (*sql.Rows, error) {
	return tx.Query(sqlSelectUserPublicKeys, userID)
}

func (me *DB) selectNumberUserPublicKeys(tx *sql.Tx, userID int) *sql.Row {
	return tx.QueryRow(sqlSelectNumberUserPublicKeys, userID)
}

func (me *DB) selectPublicKey(tx *sql.Tx, key string) *sql.Row {
	return tx.QueryRow(sqlSelectPublicKey, key)
}

func (me *DB) selectEncryptKey(tx *sql.Tx, publicKeyID int, globalID string) *sql.Row {
	return tx.QueryRow(sqlSelectEncryptKey, publicKeyID, globalID)
}

func (me *DB) selectEncryptKeys(tx *sql.Tx, publicKeyID int) (*sql.Rows, error) {
	return tx.Query(sqlSelectEncryptKeys, publicKeyID)
}

func (me *DB) selectNews(tx *sql.Tx, id int) *sql.Row {
	return tx.QueryRow(sqlSelectNews, id)
}

func (me *DB) selectNewsList(tx *sql.Tx, tag string, offset int) (*sql.Rows, error) {
	return tx.Query(sqlSelectNewsList, tag, offset)
}

func (me *DB) deleteUserPublicKey(tx *sql.Tx, userID int, publicKey string) error {
	_, err := tx.Exec(sqlDeleteUserPublicKey, userID, publicKey)
	return err
}

func (me *DB) deleteUser(tx *sql.Tx, userID int) error {
	_, err := tx.Exec(sqlDeleteUser, userID)
	return err
}

func (me *DB) deleteToken(tx *sql.Tx, token string) error {
	_, err := tx.Exec(sqlDeleteToken, token)
	return err
}

func (me *DB) updateMergePublicKeys(tx *sql.Tx, userID1 int, userID2 int) error {
	_, err := tx.Exec(sqlUpdateMergePublicKeys, userID1, userID2)
	return err
}

func (me *DB) createUserTable(tx *sql.Tx) error {
	_, err := tx.Exec(sqlCreateUserTable)
	return err
}

func (me *DB) createPublicKeyTable(tx *sql.Tx) error {
	_, err := tx.Exec(sqlCreatePublicKeyTable)
	return err
}

func (me *DB) createEncryptKeyTable(tx *sql.Tx) error {
	_, err := tx.Exec(sqlCreateEncryptKeyTable)
	return err
}

func (me *DB) createNamedSeqTable(tx *sql.Tx) error {
	_, err := tx.Exec(sqlCreateNamedSeqTable)
	return err
}

func (me *DB) createNewsTable(tx *sql.Tx) error {
	_, err := tx.Exec(sqlCreateNewsTable)
	return err
}

func (me *DB) createNewsTagTable(tx *sql.Tx) error {
	_, err := tx.Exec(sqlCreateNewsTagTable)
	return err
}

func (me *DB) createTokenTable(tx *sql.Tx) error {
	_, err := tx.Exec(sqlCreateTokenTable)
	return err
}

func (me *DB) scanUser(r *sql.Row) (*charm.User, error) {
	u := &charm.User{}
	var un, ue, ub sql.NullString
	var ca sql.NullTime
	err := r.Scan(&u.ID, &u.CharmID, &un, &ue, &ub, &ca)
	if err != nil {
		return nil, err
	}
	if un.Valid {
		u.Name = un.String
	}
	if ue.Valid {
		u.Email = ue.String
	}
	if ub.Valid {
		u.Bio = ub.String
	}
	if ca.Valid {
		u.CreatedAt = &ca.Time
	}
	return u, nil
}

// WrapTransaction runs the given function within a transaction.
func (me *DB) WrapTransaction(f func(tx *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("error starting transaction", "err", err)
		return err
	}
	for {
		err = f(tx)
		if err != nil {
			serr, ok := err.(*sqlite.Error)
			if ok && serr.Code() == sqlitelib.SQLITE_BUSY {
				continue
			}
			log.Error("error in transaction", "err", err)
			return err
		}
		err = tx.Commit()
		if err != nil {
			log.Error("error committing transaction", "err", err)
			return err
		}
		break
	}
	return nil
}
