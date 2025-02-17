package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("secretkey")

func createJWT(email string) (string, error) {

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": email,
		"iss": "yapper",
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(),
		"iat": time.Now().Unix(),
	})

	jwt, err := claims.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return jwt, nil
}

func VerifyJWT(authHeader string) (jwt.MapClaims, error) {
	// Validate header format: "Bearer <token>"
	splitToken := strings.Split(authHeader, "Bearer ")
	if len(splitToken) != 2 {
		return nil, errors.New("Invalid authorization format")
	}
	tokenString := splitToken[1]

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("Invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims format")
	}

	if iss, ok := claims["iss"].(string); !ok || iss != "yapper" {
		return nil, fmt.Errorf("invalid token issuer")
	}

	if _, ok := claims["sub"].(string); !ok {
		return nil, fmt.Errorf("token missing subject")
	}

	return claims, nil
}
