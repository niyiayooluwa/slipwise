package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mustConnectDB opens the pgx pool used by every domain's repository
// layer and confirms it's actually reachable with a ping before
// handing it back — a pool that connects lazily can otherwise mask a
// bad DATABASE_URL until the first real query fails deep in a
// request. Exits on failure since the server is useless without a DB.
func mustConnectDB(databaseURL string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	slog.Info("connected to database successfully")

	return pool
}
