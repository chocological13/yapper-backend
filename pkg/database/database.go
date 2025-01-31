package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(connString string) *pgxpool.Pool {
	dbpool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		panic(err)
	}

	return dbpool
}
