// wiring.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mustGetEnv reads a required environment variable or exits — fine
// for a handful of startup secrets, swap for internal/config once
// there are more than a couple of these.
func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}

// mustConnectDB opens the pgx pool used by every domain's repository
// layer and confirms it's actually reachable with a ping before
// handing it back — a pool that connects lazily can otherwise mask a
// bad DATABASE_URL until the first real query fails deep in a
// request. Exits on failure since the server is useless without a DB.
func mustConnectDB() *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), mustGetEnv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	fmt.Println("✅ connected to database successfully")

	return pool
}