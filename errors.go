package iris

import "errors"

var (
	ErrMalformedMsg        = errors.New("malformed message")
	ErrInvalidCode         = errors.New("invalid message code")
	ErrServiceNotAvailable = errors.New("service not available")
	ErrDisconnect          = errors.New("disconnect code received")
	ErrConnectionClose     = errors.New("connection closed")
	ErrContextDone         = errors.New("context done")
)
