package errorsx

import (
	"errors"
)

var (
	ErrClosed         = errors.New("resource closed")
	ErrNotImplemented = errors.New("not implemented")
	ErrInternal       = errors.New("internal error")
)
