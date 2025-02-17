package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/chocological13/yapper-backend/pkg/auth"
	"github.com/redis/go-redis/v9"
)

type Middleware func(next http.Handler) http.Handler

func Auth(rdb *redis.Client) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			blacklisted, err := rdb.Exists(r.Context(), "jwt:blacklist:"+authHeader).Result()
			if err != nil {
				http.Error(w, "Error checking token status", http.StatusInternalServerError)
				return
			}
			if blacklisted > 0 {
				http.Error(w, "Token has been revoked", http.StatusUnauthorized)
				return
			}

			claims, err := auth.VerifyJWT(authHeader)
			if err != nil {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
			email := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), "sub", email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Log requests
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func LogRequests(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			logger.Info("request completed",
				"method", r.Method,
				"uri", r.RequestURI,
				"status", rw.status,
				"duration", time.Since(start))
		})
	}
}
