package errorsx

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthorized    = errors.New("auth required")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrForbidden       = errors.New("forbidden")
	ErrAlreadyExists   = errors.New("already exists")
	ErrRequired        = errors.New("required value is missing")
	ErrClosed          = errors.New("resource closed")
	ErrNotImplemented  = errors.New("not implemented")
	ErrInternal        = errors.New("internal error")
)
