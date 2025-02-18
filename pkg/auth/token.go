package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

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

const (
	TokenLength          = 32 // 32 bytes = 64 hex characters
	ForgotPasswordExpiry = 60 * time.Minute
	EmailChangeExpiry    = 60 * time.Minute
)

func generateToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating tokens: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func getExistingToken(ctx context.Context, rdb *redis.Client, tokenType TokenType, userEmail string) (string, error) {
	userKey := fmt.Sprintf("%s:%s", tokenType, userEmail)
	existingToken, err := rdb.Get(ctx, userKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return existingToken, nil
}

func CreateToken(ctx context.Context, rdb *redis.Client, tokenType TokenType, userEmail string,
	extraData string) (*TokenDetails, error) {

	// Check for existing token first
	existingToken, err := getExistingToken(ctx, rdb, tokenType, userEmail)
	if err != nil {
		return nil, err
	}
	if existingToken != "" {
		return ValidateToken(ctx, rdb, tokenType, existingToken)
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating tokens: %w", err)
	}

	var expiry time.Duration
	switch tokenType {
	case ForgotPassword:
		expiry = ForgotPasswordExpiry
	case EmailChange:
		expiry = EmailChangeExpiry
	}

	expiresAt := time.Now().Add(expiry)

	details := &TokenDetails{
		Token:     token,
		Type:      tokenType,
		UserEmail: userEmail,
		ExtraData: extraData,
		ExpiresAt: expiresAt,
	}

	// Store token in redis
	key := fmt.Sprintf("%s:%s", tokenType, token)
	value := fmt.Sprintf("%s:%s", userEmail, extraData)

	err = rdb.Set(ctx, key, value, expiry).Err()
	if err != nil {
		return nil, fmt.Errorf("saving tokens: %w", err)
	}

	// Store token for initial user lookup
	userKey := fmt.Sprintf("%s:%s", tokenType, userEmail)
	err = rdb.Set(ctx, userKey, token, expiry).Err()
	if err != nil {
		return nil, fmt.Errorf("saving tokens: %w", err)
	}

	return details, nil
}

func ValidateToken(ctx context.Context, rdb *redis.Client, tokenType TokenType, token string) (*TokenDetails, error) {
	key := fmt.Sprintf("%s:%s", tokenType, token)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("getting tokens: %w", err)
	}

	var userEmail, extraData string
	parts := strings.SplitN(value, ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid tokens format: %s", value)
	}
	userEmail, extraData = parts[0], parts[1]

	return &TokenDetails{
		Token:     token,
		Type:      tokenType,
		UserEmail: userEmail,
		ExtraData: extraData,
	}, nil

}

func DeleteToken(ctx context.Context, rdb *redis.Client, tokenType TokenType, token, userEmail string) error {
	key := fmt.Sprintf("%s:%s", tokenType, token)
	err := rdb.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("deleting tokens: %w", err)
	}
	userKey := fmt.Sprintf("%s:%s", tokenType, userEmail)
	err = rdb.Del(ctx, userKey).Err()
	if err != nil {
		return fmt.Errorf("deleting tokens: %w", err)
	}
	return nil
}
