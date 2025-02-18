package main

import (
	"context"
	"github.com/chocological13/yapper-backend/pkg/mediaservice"
	"log"
	"log/slog"
	"os"

	"github.com/chocological13/yapper-backend/pkg/api"
	"github.com/chocological13/yapper-backend/pkg/database"
	"github.com/joho/godotenv"
)

var ctx = context.Background()

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dbpool := database.ConnectDB(os.Getenv("DATABASE_URL"))
	defer dbpool.Close()
	logger.Info("database connection pool established")

	rdb := database.ConnectRedis()
	logger.Info("redis connection established")

	mediaCfg := mediaservice.LoadConfigFromEnv()
	mediaService, err := mediaservice.NewMediaService(mediaCfg)
	if err != nil {
		logger.Error("failed to initiate media service", "error", err)
		os.Exit(1)
	}
	logger.Info("media service initialized")

	api.StartServer(dbpool, rdb, mediaService)
}
