package apperrors

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrDuplicateUsername  = errors.New("username already exists")
	ErrContextNotFound    = errors.New("context not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
