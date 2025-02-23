package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/chocological13/yapper-backend/pkg/auth"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getUser(ctx context.Context, dbpool *pgxpool.Pool, req GetUserRequest) (*User, error) {
	user, err := repository.New(dbpool).GetUser(ctx, repository.GetUserParams{
		UserID:   req.UserID,
		Username: req.Username,
		Email:    req.Email,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return mapUserFromDB(user), nil
}

func getCurrentUser(ctx context.Context, dbpool *pgxpool.Pool) (*User, error) {
	email, ok := ctx.Value("sub").(string)
	if !ok {
		return nil, apperrors.ErrContextNotFound
	}

	return getUser(ctx, dbpool, GetUserRequest{Email: email})
}

func updateUser(ctx context.Context, dbpool *pgxpool.Pool, req UpdateUserRequest) (*User, error) {
	user, err := getCurrentUser(ctx, dbpool)
	if err != nil {
		return nil, err
	}

	updatedUser, err := repository.New(dbpool).UpdateUser(ctx, repository.UpdateUserParams{
		UserID:   user.ID,
		Username: req.Username,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_username_key" {
			return nil, apperrors.ErrDuplicateUsername
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return mapUserFromDB(updatedUser), err
}

func deleteUser(ctx context.Context, dbpool *pgxpool.Pool, req DeleteUserRequest) error {
	// TODO : add more confirmation with email confirmation

	user, err := getCurrentUser(ctx, dbpool)
	if err != nil {
		return err
	}

	match, _, err := auth.VerifyPassword(req.Password, user.Password)
	if !match {
		return apperrors.ErrInvalidCredentials
	}

	err = repository.New(dbpool).DeleteUser(ctx, user.ID)
	return err
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
