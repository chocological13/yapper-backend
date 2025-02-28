package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/chocological13/yapper-backend/pkg/media"

	"github.com/chocological13/yapper-backend/pkg/api"
	"github.com/chocological13/yapper-backend/pkg/database"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	dbpool := database.ConnectDB(os.Getenv("DATABASE_URL"))
	defer dbpool.Close()
	logger.Info("database connection pool established")

	rdb := database.ConnectRedis()
	logger.Info("redis connection established")

	storageCfg := media.LoadConfigFromEnv()
	storageService, err := media.NewStorageService(storageCfg)
	if err != nil {
		logger.Error("failed to initiate media service", "error", err)
		os.Exit(1)
	}
	logger.Info("storage service initialized")

	api.StartServer(dbpool, rdb, storageService, logger)
}
