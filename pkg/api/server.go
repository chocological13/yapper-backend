package api

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/chocological13/yapper-backend/pkg/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "0.1.0"

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

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := &app{
		cfg,
		logger,
		dbpool,
	}

	authAPI := auth.New(app.dbpool)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /register", authAPI.RegisterUser)
	mux.HandleFunc("POST /login", authAPI.LoginUser)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	err := srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}
