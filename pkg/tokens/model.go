package tokens

import "time"

type TokenType string

const (
	ForgotPassword TokenType = "forgot_password"
	EmailChange    TokenType = "email_change"
)

type TokenDetails struct {
	Token     string
	Type      TokenType
	UserEmail string
	ExtraData string
	ExpiresAt time.Time
}
