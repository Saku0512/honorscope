package honorscope

import (
	"errors"
	"fmt"
)

// Sentinel errors. Use errors.Is. HTTP failures also return *RequestError.
var (
	ErrUserNotFound    = errors.New("github user not found")
	ErrRateLimited     = errors.New("github rate limit exceeded")
	ErrInvalidUsername = errors.New("invalid github username")
)

// RequestError is an HTTP failure while loading a public profile.
type RequestError struct {
	URL        string
	StatusCode int
	Err        error
}

func (e *RequestError) Error() string {
	if e == nil {
		return "honorscope: request failed"
	}
	if e.Err != nil {
		return fmt.Sprintf("honorscope: GET %s: %v", e.URL, e.Err)
	}
	return fmt.Sprintf("honorscope: GET %s: HTTP %d", e.URL, e.StatusCode)
}

func (e *RequestError) Unwrap() error { return e.Err }
