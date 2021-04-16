// Package proto contains structs used for client/server communication.
package proto

// Message is used as a wrapper for simple client/server messages.
type Message struct {
	Message string `json:"message"`
}
