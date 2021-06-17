package sqlite

import (
	"database/sql"
	"fmt"
	"log"
)

const (
	sqlCreateStatsTable = `CREATE TABLE IF NOT EXISTS stats(
	id INTEGER PRIMARY KEY,
	api_link_gen integer NOT NULL DEFAULT 0,
	api_link_request integer NOT NULL DEFAULT 0,
	api_unlink integer NOT NULL DEFAULT 0,
	api_auth integer NOT NULL DEFAULT 0,
	api_keys integer NOT NULL DEFAULT 0,
	link_gen integer NOT NULL DEFAULT 0,
	link_request integer NOT NULL DEFAULT 0,
	get_keys integer NOT NULL DEFAULT 0,
	get_id integer NOT NULL DEFAULT 0,
	get_jwt integer NOT NULL DEFAULT 0,
	get_user_by_id integer NOT NULL DEFAULT 0,
	get_user integer NOT NULL DEFAULT 0,
	set_user_name integer NOT NULL DEFAULT 0,
	get_news integer NOT NULL DEFAULT 0,
	post_news integer NOT NULL DEFAULT 0,
	get_news_list integer NOT NULL DEFAULT 0,
	created_at timestamp default current_timestamp
	)`
)

// Stats implements the stats.Stats interface for SQLite
type Stats struct {
	db *sql.DB
}

// NewStats returns a *Stats with the default configuration.
func NewStats(path string) (*Stats, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Stats{db: db}
	err = s.createDB()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Stats) createDB() error {
	_, err := s.db.Exec(sqlCreateStatsTable)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO stats (api_link_gen) VALUES(0)")
	return err
}

func (s *Stats) increment(field string) {
	// SQLite doesn't use placeholders for table or field names
	stmt := fmt.Sprintf("UPDATE stats SET %s = %s+1 WHERE id = (SELECT MAX(id) from stats)", field, field)
	_, err := s.db.Exec(stmt)
	if err != nil {
		log.Printf("error updating stat '%s': %s", field, err)
	}
}

// APILinkGen increments the number of api-link-gen calls.
func (s *Stats) APILinkGen() {
	s.increment("api_link_gen")
}

// APILinkRequest increments the number of api-link-request calls.
func (s *Stats) APILinkRequest() {
	s.increment("api_link_request")
}

// APIUnlink increments the number of api-unlink calls.
func (s *Stats) APIUnlink() {
	s.increment("api_unlink")
}

// APIAuth increments the number of api-auth calls.
func (s *Stats) APIAuth() {
	s.increment("api_auth")
}

// APIKeys increments the number of api-keys calls.
func (s *Stats) APIKeys() {
	s.increment("api_keys")
}

// LinkGen increments the number of link-gen calls.
func (s *Stats) LinkGen() {
	s.increment("link_gen")
}

// LinkRequest increments the number of link-request calls.
func (s *Stats) LinkRequest() {
	s.increment("link_request")
}

// Keys increments the number of keys calls.
func (s *Stats) Keys() {
	s.increment("get_keys")
}

// ID increments the number of id calls.
func (s *Stats) ID() {
	s.increment("get_id")
}

// JWT increments the number of jwt calls.
func (s *Stats) JWT() {
	s.increment("get_jwt")
}

// GetUserByID increments the number of user-by-id calls.
func (s *Stats) GetUserByID() {
	s.increment("get_user_by_id")
}

// GetUser increments the number of get-user calls.
func (s *Stats) GetUser() {
	s.increment("get_user")
}

// SetUserName increments the number of set-user-name calls.
func (s *Stats) SetUserName() {
	s.increment("set_user_name")
}

// GetNews increments the number of get-news calls.
func (s *Stats) GetNews() {
	s.increment("get_news")
}

// PostNews increments the number of post-news calls.
func (s *Stats) PostNews() {
	s.increment("post_news")
}

// GetNewsList increments the number of get-news-list calls.
func (s *Stats) GetNewsList() {
	s.increment("get_news_list")
}
