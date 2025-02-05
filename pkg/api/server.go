package api

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/chocological13/yapper-backend/pkg/api/middleware"
	"github.com/chocological13/yapper-backend/pkg/database/repository"
	"github.com/chocological13/yapper-backend/pkg/users"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
}

func StartServer(dbpool *pgxpool.Pool) {
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
	}

	authAPI := auth.New(app.dbpool)

	queries := repository.New(app.dbpool)
	userService := users.NewUserService(queries)
	userHandler := users.NewUserHandler(userService)

	mux := http.NewServeMux()

	// Public routes
	// Auth routes
	mux.HandleFunc("POST "+apiVersion+"/register", authAPI.RegisterUser)
	mux.HandleFunc("POST "+apiVersion+"/login", authAPI.LoginUser)

	// Testing purposes
	mux.HandleFunc("GET "+apiVersion+"/users", userHandler.GetUser)

	// Protected routes (auth required)
	// Users
	mux.Handle("GET "+apiVersion+"/users/me", middleware.Auth(http.HandlerFunc(userHandler.GetCurrentUser)))
	mux.Handle("PUT "+apiVersion+"/users/me", middleware.Auth(http.HandlerFunc(userHandler.UpdateUser)))
	mux.Handle("PUT "+apiVersion+"/users/me/email", middleware.Auth(http.HandlerFunc(userHandler.UpdateUserEmail)))

	// Add future middleware here
	muxWithMiddleware := middleware.LogRequests(logger)(mux)

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
