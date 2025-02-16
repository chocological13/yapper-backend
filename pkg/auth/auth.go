package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/jackc/pgx/v5"
	"time"

	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func register(ctx context.Context, dbpool *pgxpool.Pool, rdb *redis.Client, p *AuthInput) (string, error) {
	passwordHash, err := HashPassword(p.Password)
	if err != nil {
		return "", err
	}

	params := repository.NewUserParams{
		Email:    p.Email,
		Username: p.Username,
		Password: passwordHash,
	}

	user, err := repository.New(dbpool).NewUser(ctx, params)
	if err != nil {
		return "", err
	}

	jwt, err := createJWT(user)
	if err != nil {
		return "", apperrors.ErrJWTGenerationError
	}

	err = rdb.Set(ctx, fmt.Sprintf("jwt:%s", p.Email), jwt, 7*24*time.Hour).Err()
	if err != nil {
		return "", apperrors.ErrJWTGenerationError
	}

	return jwt, nil
}

func login(ctx context.Context, dbpool *pgxpool.Pool, rdb *redis.Client, p *AuthInput) (string, error) {
	params := repository.GetUserParams{
		Email:    p.Email,
		Username: p.Username,
	}

	user, err := repository.New(dbpool).GetUser(ctx, params)
	if err != nil {
		return "", err
	}

	match, _, err := VerifyPassword(p.Password, user.Password)
	if err != nil {
		return "", err
	}

	if !match {
		return "", apperrors.ErrInvalidCredentials
	}

	val, _ := rdb.Get(ctx, fmt.Sprintf("jwt:%s", p.Email)).Result()
	if val != "" {
		token := fmt.Sprintf("Bearer %s", val)
		isBlacklisted, _ := rdb.Get(ctx, fmt.Sprintf("jwt:blacklist:%s", token)).Result()
		if isBlacklisted != "" {
			val = ""
		}
	}

	if val == "" {
		newJWT := ""
		newJWT, err = createJWT(user.Email)
		if err != nil {
			return "", apperrors.ErrJWTGenerationError
		}

		err = rdb.Set(ctx, fmt.Sprintf("jwt:%s", p.Email), newJWT, 7*24*time.Hour).Err()
		if err != nil {
			return "", apperrors.ErrJWTGenerationError
		}

		return newJWT, nil
	}

	jwt, err := createJWT(user.Email)
	if err != nil {
		return "", apperrors.ErrJWTGenerationError
	}

	err = rdb.Set(ctx, fmt.Sprintf("jwt:%s", p.Email), jwt, 7*24*time.Hour).Err()
	if err != nil {
		return "", apperrors.ErrJWTGenerationError
	}

	return jwt, nil
}

// forgorPassword currently provides a skeleton implementation of the overall forgot password functionality.
// This is for logged-out users who can't access their account
func forgorPassword(ctx context.Context, dbpool *pgxpool.Pool, p *ForgotPasswordRequest) error {
	user, err := repository.New(dbpool).GetUser(ctx, repository.GetUserParams{
		Email:    p.Email,
		Username: "",
	})
	if err != nil {
		return fmt.Errorf("failed to get user: %w", apperrors.ErrUserNotFound)
	}

	hashedPassword, err := HashPassword(p.NewPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	err = repository.New(dbpool).UpdatePassword(ctx, repository.UpdatePasswordParams{
		UserID:   user.UserID,
		Password: hashedPassword,
	})
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	return nil
}

// resetPassword currently provides a skeleton implementation of the overall forgot password functionality.
// This is for logged-in users who want to update their passwords
func resetPassword(ctx context.Context, dbpool *pgxpool.Pool, p *ResetPasswordRequest) error {
	user, err := getCurrentUser(ctx, dbpool)
	if err != nil {
		return err
	}

	match, _, err := VerifyPassword(p.CurrentPassword, user.Password)
	if err != nil {
		return err
	}

	if !match {
		return apperrors.ErrInvalidCredentials
	}

	hashedPassword, err := HashPassword(p.NewPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	err = repository.New(dbpool).UpdatePassword(ctx, repository.UpdatePasswordParams{
		UserID:   user.UserID,
		Password: hashedPassword,
	})
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}

	return nil
}

func updateEmail(ctx context.Context, dbpool *pgxpool.Pool, p *UpdateEmailRequest) error {
	user, err := getCurrentUser(ctx, dbpool)
	if err != nil {
		return err
	}

	match, _, err := VerifyPassword(p.Password, user.Password)
	if err != nil {
		return err
	}

	if !match {
		return apperrors.ErrInvalidCredentials
	}

	var exist bool
	exist, err = checkUserExists(ctx, dbpool, p.NewEmail)
	if err != nil {
		return err
	}
	if exist {
		return apperrors.ErrDuplicateEmail
	}

	_, err = repository.New(dbpool).UpdateEmail(ctx, repository.UpdateEmailParams{
		UserID: user.UserID,
		Email:  p.NewEmail,
	})

	return err
}

func getCurrentUser(ctx context.Context, dbpool *pgxpool.Pool) (*repository.User, error) {
	var email string
	var ok bool
	if email, ok = ctx.Value("sub").(string); email == "" || !ok {
		return nil, apperrors.ErrContextNotFound
	}

	user, err := repository.New(dbpool).GetUser(ctx, repository.GetUserParams{
		Email: email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func checkUserExists(ctx context.Context, dbpool *pgxpool.Pool, email string) (bool, error) {
	user, err := repository.New(dbpool).GetUser(ctx, repository.GetUserParams{
		Email: email,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("error checking user existence: %w", err)
	}
	return &user != nil, nil

}

func invalidateJwt(ctx context.Context, rdb *redis.Client, email string) error {
	token, err := rdb.Get(ctx, fmt.Sprintf("jwt:%s", email)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return apperrors.ErrTokenNotFound
	}

	if token != "" {
		err = rdb.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", token), "true", 7*24*time.Hour).Err()
		if err != nil {
			return err
		}
	}

	return nil
}
