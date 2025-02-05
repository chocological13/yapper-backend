package users

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/auth"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrContextNotFound   = errors.New("could not get current user from context")
	ErrDuplicateEmail    = errors.New("email already exists")
	ErrDuplicateUsername = errors.New("username already exists")
	ErrInvalidPassword   = errors.New("invalid password")

	// TODO Future: Add these when implementing email verification
	// ErrEmailNotVerified  = errors.New("email not verified")
	// ErrTokenExpired      = errors.New("verification token expired")
)

type UserService struct {
	repository *repository.Queries

	// TODO services to implement for verification purposes
	// emailService EmailService
	// tokenService TokenService
}

func NewUserService(repository *repository.Queries) *UserService {
	return &UserService{repository: repository}
}

func (s *UserService) GetUser(ctx context.Context, req GetUserRequest) (*User, error) {
	user, err := s.repository.GetUser(ctx, repository.GetUserParams{
		UserID:   req.UserID,
		Username: req.Username,
		Email:    req.Email,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return mapUserFromDB(user), nil
}

func (s *UserService) GetCurrentUser(ctx context.Context) (*User, error) {
	email, ok := ctx.Value("sub").(string)
	if !ok {
		return nil, ErrContextNotFound
	}

	return s.GetUser(ctx, GetUserRequest{Email: email})
}

func (s *UserService) UpdateUser(ctx context.Context, req UpdateUserRequest) (*User, error) {
	user, err := s.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	updatedUser, err := s.repository.UpdateUser(ctx, repository.UpdateUserParams{
		UserID:   user.ID,
		Username: req.Username,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_username_key" {
			return nil, ErrDuplicateUsername
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return mapUserFromDB(updatedUser), err
}

// TODO: Complete implementation of email verification flow
// 1. Make service to generate verification token
// 2. Store new email in the token (if with jwt) or with
// 3. Send verification email
// 4. Store pending email change
// 5. Confirm email, verify token
// 6. Update user

// UpdateEmail currently provides a skeleton implementation of the overall update email
// functionality. TODO : complete after mailer service is established
func (s *UserService) UpdateEmail(ctx context.Context, req UpdateEmailRequest) (*User, error) {
	user, err := s.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	// Verify current password as a layer of security for now
	match, _, err := auth.VerifyPassword(req.Password, user.Password)
	if err != nil {
		return nil, fmt.Errorf("verifying password: %w", err)
	}
	if !match {
		return nil, ErrInvalidPassword
	}

	updatedUser, err := s.repository.UpdateEmail(ctx, repository.UpdateEmailParams{
		UserID: user.ID,
		Email:  req.NewEmail,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_email_key" {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("updating email: %w", err)
	}

	return mapUserFromDB(updatedUser), nil
}

// Helper function to map database user to domain user
func mapUserFromDB(dbUser repository.User) *User {
	return &User{
		ID:        dbUser.UserID,
		Username:  dbUser.Username,
		Email:     dbUser.Email,
		Password:  dbUser.Password,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: mapTimestamptz(dbUser.UpdatedAt),
		DeletedAt: mapTimestamptz(dbUser.DeletedAt),
	}
}

// Helper function to handle nullable timestamps
func mapTimestamptz(t pgtype.Timestamptz) *pgtype.Timestamptz {
	if !t.Valid {
		return nil
	}
	return &t
}
