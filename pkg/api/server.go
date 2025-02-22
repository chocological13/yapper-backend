package api

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chocological13/yapper-backend/pkg/api/middleware"
	"github.com/chocological13/yapper-backend/pkg/users"
	"github.com/redis/go-redis/v9"

	"github.com/chocological13/yapper-backend/pkg/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "0.1.0"

const apiVersion = "/api/v1"

type config struct {
	port int
	env  string
}

type app struct {
	cfg    config
	logger *slog.Logger
	dbpool *pgxpool.Pool
	rdb    *redis.Client
}

func StartServer(dbpool *pgxpool.Pool, rdb *redis.Client) {
	var cfg config

	flag.IntVar(&cfg.port, "port", 8080, "API server port")
	flag.StringVar(&cfg.env, "env", "dev", "Environment (dev|staging|prod)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	app := &app{
		cfg,
		logger,
		dbpool,
		rdb,
	}

	authAPI := auth.New(app.dbpool, app.rdb)
	userHandler := users.NewUserHandler(app.dbpool)

	mux := http.NewServeMux()

	// Public routes
	// Auth routes
	mux.HandleFunc("POST /register", authAPI.RegisterUser)
	mux.HandleFunc("POST /login", authAPI.LoginUser)
	mux.Handle("POST /logout", middleware.Auth(app.rdb)(http.HandlerFunc(authAPI.LogoutUser)))

	// Users routes
	mux.HandleFunc("POST /forgot-password", authAPI.InitiateForgotPassword)
	mux.HandleFunc("PATCH /forgot-password", authAPI.CompleteForgotPassword)

	// Testing purposes
	mux.HandleFunc("GET /users", userHandler.GetUser)

	// Protected routes (auth required)

	// Auth-related users operations
	authMux := http.NewServeMux()
	authMux.HandleFunc("POST /users/me/email", authAPI.InitiateUpdateUserEmail)
	authMux.HandleFunc("PATCH /users/me/email", authAPI.CompleteUpdateUserEmail)
	authMux.HandleFunc("PATCH /users/me/reset-password", authAPI.ResetPassword)

	// Users
	authMux.HandleFunc("GET /users/me", userHandler.GetCurrentUser)
	authMux.HandleFunc("PUT /users/me", userHandler.UpdateUser)
	authMux.HandleFunc("DELETE /users/me", userHandler.DeleteUser)

	mux.Handle("/", middleware.Auth(app.rdb)(authMux))

	// /v1 prefix
	v1 := http.NewServeMux()
	v1.Handle("/v1/", http.StripPrefix("/v1", mux))

	// Add future middleware here
	muxWithMiddleware := middleware.LogRequests(logger)(v1)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      muxWithMiddleware,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		logger.Info("shutting down server", slog.String("signal", s.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutdownError <- srv.Shutdown(ctx)
	}()

	logger.Info("starting server", "port", cfg.port, "env", cfg.env)

	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	shutdownErr := <-shutdownError
	if shutdownErr != nil {
		logger.Error("graceful shutdown failed", "error", shutdownErr)
	} else {
		logger.Info("stopped server", slog.String("addr", srv.Addr))
	}
}
