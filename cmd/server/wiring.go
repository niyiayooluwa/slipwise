package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mustConnectDB opens the pgx connection pool used by every domain's repository layer.
//
// Why pgxpool instead of standard database/sql?
//  1. Concurrency: It manages reusable PostgreSQL connections with automatic pooling.
//  2. Binary protocol: Faster serialization/deserialization than plain text SQL drivers.
//  3. Fail-fast safety: A lazy pool might start without errors only to fail on the user's
//     first HTTP request. By sending a Ping immediately, we ensure the DB is healthy before
//     the server starts accepting incoming traffic.
func mustConnectDB(databaseURL string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		// Fails loudly if connection to database cannot be established
		log.Fatalf("unable to create connection pool: %v", err)
	}

	// Immediate connectivity check: fail hard and loud during boot if PostgreSQL is down.
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	slog.Info("connected to database successfully")

	return pool
}
