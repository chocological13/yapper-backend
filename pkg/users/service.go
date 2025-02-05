package users

import (
	"context"
	"errors"
	"fmt"
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

func (s *UserService) GetUser(ctx context.Context, arg GetUserRequest) (*User, error) {
	user, err := s.repository.GetUser(ctx, repository.GetUserParams{
		UserID:   arg.UserID,
		Username: arg.Username,
		Email:    arg.Email,
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

func (s *UserService) UpdateUser(ctx context.Context, arg UpdateUserRequest) (*User, error) {
	user, err := s.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	updatedUser, err := s.repository.UpdateUser(ctx, repository.UpdateUserParams{
		UserID:   user.ID,
		Username: arg.Username,
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

// Helper function to see
