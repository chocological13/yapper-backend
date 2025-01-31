package middleware

import (
	"context"
	"net/http"

	"github.com/chocological13/yapper-backend/pkg/auth"
)

type Middleware func(next http.Handler) http.Handler

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		if claims, err := auth.VerifyJWT(authHeader); err == nil {
			email := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), "sub", email)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		}
	})
}
