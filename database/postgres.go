package database

import (
	"context"
	"fmt"
	"time"

	"eventra-api/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect membuat connection pool ke PostgreSQL dan memverifikasi koneksi.
func Connect(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config database: %w", err)
	}
	poolConfig.MaxConns = cfg.DBMaxConns

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("buat pool database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database gagal: %w", err)
	}

	return pool, nil
}
