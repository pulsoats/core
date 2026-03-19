package derrors

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthorized    = errors.New("auth required")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrAlreadyExists   = errors.New("already exists")
	ErrRequired        = errors.New("required value is missing")
)
