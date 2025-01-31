package main

import (
	"log"
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

	dbpool := database.ConnectDB(os.Getenv("DATABASE_URL"))
	defer dbpool.Close()

	api.StartServer(dbpool)
}
