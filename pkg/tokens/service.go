package tokens

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/apperrors"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

const (
	TokenLength          = 32 // 32 bytes = 64 hex characters
	ForgotPasswordExpiry = 60 * time.Minute
	EmailChangeExpiry    = 60 * time.Minute
)

type Service interface {
	CreateToken(ctx context.Context, tokenType TokenType, userEmail string, extraData string) (*TokenDetails, error)
	ValidateToken(ctx context.Context, tokenType TokenType, token string) (*TokenDetails, error)
	BlacklistToken(ctx context.Context, tokenType TokenType, token string) error
}

type tokenService struct {
	rdb *redis.Client
}

func NewTokenService(rdb *redis.Client) Service {
	return &tokenService{
		rdb: rdb,
	}
}

func (s *tokenService) generateToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating tokens: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (s *tokenService) CreateToken(ctx context.Context, tokenType TokenType, userEmail string,
	extraData string) (*TokenDetails, error) {

	token, err := s.generateToken()
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

	// Store tokens in redis
	key := fmt.Sprintf("%s:%s", tokenType, token)
	value := fmt.Sprintf("%s:%s", userEmail, extraData)

	err = s.rdb.Set(ctx, key, value, expiry).Err()
	if err != nil {
		return nil, fmt.Errorf("saving tokens: %w", err)
	}

	return details, nil
}

func (s *tokenService) ValidateToken(ctx context.Context, tokenType TokenType, token string) (*TokenDetails, error) {
	if valid := s.isTokenValid(ctx, tokenType, token); !valid {
		return nil, apperrors.ErrInvalidToken
	}

	key := fmt.Sprintf("%s:%s", tokenType, token)
	value, err := s.rdb.Get(ctx, key).Result()
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

func (s *tokenService) BlacklistToken(ctx context.Context, tokenType TokenType, token string) error {
	key := fmt.Sprintf("%s:%s:%s", tokenType, "blacklist", token)
	err := s.rdb.Set(ctx, key, true, 1*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("saving blacklisted tokens: %w", err)
	}
	return nil
}

func (s *tokenService) isTokenValid(ctx context.Context, tokenType TokenType, token string) bool {
	blacklistKey := fmt.Sprintf("%s:%s:%s", tokenType, "blacklist", token)
	exists := s.rdb.Exists(ctx, blacklistKey).Val()
	if exists != 0 {
		return false
	}
	return true
}
