package users

import (
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

type User struct {
	ID        pgtype.UUID         `json:"id"`
	Username  string              `json:"username"`
	Email     string              `json:"email"`
	Password  string              `json:"-"`
	CreatedAt pgtype.Timestamptz  `json:"created_at"`
	UpdatedAt *pgtype.Timestamptz `json:"updated_at,omitempty"`
	DeletedAt *pgtype.Timestamptz `json:"deleted_at,omitempty"`
}

type GetUserRequest struct {
	UserID   pgtype.UUID `json:"user_id"`
	Username string      `json:"username"`
	Email    string      `json:"email"`
}

type UpdateUserRequest struct {
	// Add other non-sensitive fields as needed
	Username string `json:"username,omitempty"`
}

type DeleteUserRequest struct {
	Password string `json:"password"`
}

type EmailVerificationData struct {
	UserID    pgtype.UUID `json:"user_id"`
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
}
