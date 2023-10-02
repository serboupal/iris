package iris

import "errors"

var (
	ErrContextDone = errors.New("context done")
	ErrInternal    = errors.New("internal error")
	ErrMalformed   = errors.New("malformed message")
)
