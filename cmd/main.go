package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/chocological13/yapper-backend/pkg/api"
	"github.com/chocological13/yapper-backend/pkg/database"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dbpool := database.ConnectDB(os.Getenv("DATABASE_URL"))
	defer dbpool.Close()

	logger.Info("database connection pool established")

	api.StartServer(dbpool)
}
