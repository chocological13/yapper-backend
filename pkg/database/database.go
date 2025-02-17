package database

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func ConnectDB(connString string) *pgxpool.Pool {
	dbpool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		panic(err)
	}

	return dbpool
}

func ConnectRedis() *redis.Client {
	opt, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	rdb := redis.NewClient(opt)

	return rdb
}
