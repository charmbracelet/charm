package proto

import "time"

// News entity.
type News struct {
	ID        string    `json:"id"`
	Subject   string    `json:"subject"`
	Tag       string    `json:"tag"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
