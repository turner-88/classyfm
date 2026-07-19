// Package db wires the MySQL connection pool. sqlc-generated query code will live in
// internal/db/sqlc and operate against the *sql.DB returned here.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Open creates and verifies a MySQL connection pool from a DSN. If dsn is empty it
// returns (nil, nil) so early phases can boot without a database.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, nil
	}
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(25)
	pool.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return pool, nil
}
