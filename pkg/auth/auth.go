package auth

import (
	"context"
	"errors"

	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func register(ctx context.Context, dbpool *pgxpool.Pool, p *AuthInput) (string, error) {
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
		return "", err
	}

	return jwt, nil
}

func login(ctx context.Context, dbpool *pgxpool.Pool, p *AuthInput) (string, error) {
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
		println("HWHWH")
		return "", errors.New("Invalid credentials")
	}

	jwt, err := createJWT(user.Email)
	if err != nil {
		return "", err
	}

	return jwt, nil
}
