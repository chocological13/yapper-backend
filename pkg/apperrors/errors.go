package apperrors

import "errors"

// Auth related apperrors
var (
	ErrJWTGenerationError = errors.New("Failed to generate JWT")
	ErrInvalidCredentials = errors.New("Invalid credentials")
	ErrTokenNotFound      = errors.New("Token not found")
)

// User related apperrors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrContextNotFound   = errors.New("could not get current user from context")
	ErrDuplicateEmail    = errors.New("email already exists")
	ErrDuplicateUsername = errors.New("username already exists")

	// TODO Future: Add these when implementing email verification
	// ErrEmailNotVerified  = apperrors.New("email not verified")
	// ErrTokenExpired      = apperrors.New("verification token expired")
)
