package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/chocological13/yapper-backend/pkg/tokens"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"time"

	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5/pgxpool"
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case "users_email_key":
				return "", apperrors.ErrDuplicateEmail
			case "users_username_key":
				return "", apperrors.ErrDuplicateUsername
			}
		}
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

// initiateForgorPassword currently handles the initiation of password reset process for logged-out users.
func initiateForgorPassword(ctx context.Context, dbpool *pgxpool.Pool, tokenService tokens.Service,
	p *ForgotPasswordRequest) (string, error) {
	exists, err := checkUserExists(ctx, dbpool, p.Email)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", apperrors.ErrUserNotFound
	}

	details, err := tokenService.CreateToken(ctx, tokens.ForgotPassword, p.Email, "")
	if err != nil {
		return "", err
	}

	// TODO : send tokens to email
	return details.Token, nil
}

func completeForgorPassword(ctx context.Context, dbpool *pgxpool.Pool, tokenService tokens.Service, p *CompleteForgotPassword) error {
	details, err := tokenService.ValidateToken(ctx, tokens.ForgotPassword, p.Token)
	if err != nil {
		return err
	}

	user, err := repository.New(dbpool).GetUser(ctx, repository.GetUserParams{
		Email: details.UserEmail,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.ErrUserNotFound
		}
		return err
	}

	hashedPassword, err := HashPassword(p.NewPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	err = repository.New(dbpool).UpdatePassword(ctx, repository.UpdatePasswordParams{
		UserID:   user.UserID,
		Password: hashedPassword,
	})

	// blacklist token in the end
	return tokenService.BlacklistToken(ctx, tokens.ForgotPassword, p.Token)
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

func initiateUpdateEmail(ctx context.Context, dbpool *pgxpool.Pool, tokenService tokens.Service,
	p *UpdateEmailRequest) (string, error) {
	user, err := getCurrentUser(ctx, dbpool)
	if err != nil {
		return "", err
	}

	exists, err := checkUserExists(ctx, dbpool, p.NewEmail)
	if err != nil {
		return "", err
	}
	if exists {
		return "", apperrors.ErrDuplicateEmail
	}

	match, _, err := VerifyPassword(p.Password, user.Password)
	if err != nil {
		return "", err
	}

	if !match {
		return "", apperrors.ErrInvalidCredentials
	}

	tokenDetails, err := tokenService.CreateToken(ctx, tokens.EmailChange, user.Email, p.NewEmail)
	if err != nil {
		return "", err
	}

	// TODO : send token to new email for verification

	return tokenDetails.Token, nil
}

func completeUpdateUserEmail(ctx context.Context, dbpool *pgxpool.Pool, tokenService tokens.Service, p *CompleteUpdateEmail) error {
	tokenDetails, err := tokenService.ValidateToken(ctx, tokens.EmailChange, p.Token)
	if err != nil {
		return err
	}

	user, err := repository.New(dbpool).GetUser(ctx, repository.GetUserParams{
		Email: tokenDetails.UserEmail,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.ErrUserNotFound
		}
		return err
	}

	_, err = repository.New(dbpool).UpdateEmail(ctx, repository.UpdateEmailParams{
		UserID: user.UserID,
		Email:  tokenDetails.ExtraData,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "users_email_key" {
			return apperrors.ErrDuplicateEmail
		}
		return err
	}

	return tokenService.BlacklistToken(ctx, tokens.EmailChange, tokenDetails.Token)
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
