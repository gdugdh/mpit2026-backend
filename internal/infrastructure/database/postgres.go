package database

import (
	"context"
	"fmt"
	"time"

	"github.com/gdugdh24/mpit2026-backend/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// NewPostgresDB creates a new PostgreSQL database connection using sqlx
func NewPostgresDB(cfg *config.DatabaseConfig) (*sqlx.DB, error) {
	dsn := cfg.GetDSN()

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
