package apperrors

import "errors"

// Auth related apperrors
var (
	ErrJWTGenerationError = errors.New("failed to generate JWT")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenNotFound      = errors.New("token not found")
)

// User related apperrors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrContextNotFound   = errors.New("could not get current user from context")
	ErrDuplicateEmail    = errors.New("email already exists")
	ErrDuplicateUsername = errors.New("username already exists")
)

// Tokens related apperrors
var (
	ErrInvalidToken = errors.New("invalid token")
)
