package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

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
