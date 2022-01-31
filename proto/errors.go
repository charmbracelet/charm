package proto

import (
	"errors"
	"fmt"
)

// ErrMalformedKey parsing error for bad ssh key.
var ErrMalformedKey = errors.New("malformed key; is it missing the algorithm type at the beginning?")

// ErrMissingSSHAuth is used when the user is missing SSH credentials.
var ErrMissingSSHAuth = errors.New("missing ssh auth")

// ErrNameTaken is used when a user attempts to set a username and that
// username is already taken.
var ErrNameTaken = errors.New("name already taken")

// ErrNameInvalid is used when a username is invalid.
var ErrNameInvalid = errors.New("invalid name")

// ErrCouldNotUnlinkKey is used when a key can't be deleted.
var ErrCouldNotUnlinkKey = errors.New("could not unlink key")

// ErrMissingUser is used when no user record is found.
var ErrMissingUser = errors.New("no user found")

// ErrUserExists is used when attempting to create a user with an existing
// global id.
var ErrUserExists = errors.New("user already exists for that key")

// ErrPageOutOfBounds is an error for an invalid page number.
var ErrPageOutOfBounds = errors.New("page must be a value of 1 or greater")

// ErrAuthFailed indicates an authentication failure. The underlying error is
// wrapped.
type ErrAuthFailed struct {
	Err error
}

// Error returns the boxed error string.
func (e ErrAuthFailed) Error() string { return fmt.Sprintf("authentication failed: %s", e.Err.Error()) }

// Unwrap returns the boxed error.
func (e ErrAuthFailed) Unwrap() error { return e.Err }
