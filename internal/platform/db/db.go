// Package db builds and manages the Postgres connection pool.
package db

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is the subset of settings needed to open a pool.
type Config struct {
	// URL is the pooled connection string.
	URL string
	// MaxConns is the maximum pool size.
	MaxConns int32
	// MinConns is the minimum pool size kept warm.
	MinConns int32
	// StatementTimeout caps every statement.
	StatementTimeout time.Duration
	// ApplicationName is reported in pg_stat_activity.
	ApplicationName string
}

// Open creates a pool, applies the settings and verifies connectivity.
func Open(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(cfg.StatementTimeout.Milliseconds(), 10)
	if cfg.ApplicationName != "" {
		poolCfg.ConnConfig.RuntimeParams["application_name"] = cfg.ApplicationName
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
