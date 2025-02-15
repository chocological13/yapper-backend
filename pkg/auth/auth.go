package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrJWTGenerationError = errors.New("Failed to generate JWT")
	ErrInvalidCredentials = errors.New("Invalid credentials")
	ErrTokenNotFound      = errors.New("Token not found")
)

func register(ctx context.Context, dbpool *pgxpool.Pool, rdb *redis.Client, p *AuthInput) (string, error) {
	password_hash, err := HashPassword(p.Password)
	if err != nil {
		return "", err
	}

	params := repository.NewUserParams{
		Email:    p.Email,
		Username: p.Username,
		Password: password_hash,
	}

	user, err := repository.New(dbpool).NewUser(ctx, params)
	if err != nil {
		return "", err
	}

	jwt, err := createJWT(user)
	if err != nil {
		return "", ErrJWTGenerationError
	}

	err = rdb.Set(ctx, fmt.Sprintf("jwt:%s", p.Email), jwt, 7*24*time.Hour).Err()
	if err != nil {
		return "", ErrJWTGenerationError
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
		return "", ErrInvalidCredentials
	}

	val, _ := rdb.Get(ctx, fmt.Sprintf("jwt:%s", p.Email)).Result()
	if val != "" {
		return val, nil
	}

	jwt, err := createJWT(user.Email)
	if err != nil {
		return "", ErrJWTGenerationError
	}

	err = rdb.Set(ctx, fmt.Sprintf("jwt:%s", p.Email), jwt, 7*24*time.Hour).Err()
	if err != nil {
		return "", ErrJWTGenerationError
	}

	return jwt, nil
}

// ForgorPassword currently provides a skeleton implementation of the overall forgot password functionality.
// This is for logged-out users who can't access their account
func forgorPassword(ctx context.Context, dbpool *pgxpool.Pool, p *ForgotPasswordInput) error {
	user, err := repository.New(dbpool).GetUser(ctx, repository.GetUserParams{
		Email:    p.Email,
		Username: "",
	})
	if err != nil {
		return fmt.Errorf("Failed to get user: %w", err)
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

func invalidateJwt(ctx context.Context, rdb *redis.Client, email string) error {
	token, err := rdb.Get(ctx, fmt.Sprintf("jwt:%s", email)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return ErrTokenNotFound
	}

	if token != "" {
		err = rdb.Set(ctx, fmt.Sprintf("jwt:blacklist:"), token, 7*24*time.Hour).Err()
		if err != nil {
			return err
		}
	}

	return nil
}
