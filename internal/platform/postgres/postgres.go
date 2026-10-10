package postgres

import (
	"context"
	"time"

	"github.com/fisdemire/Hotel-Control/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pingTimeout = 5 * time.Second

func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, err
	}

	pc.MaxConns = cfg.MaxConns
	pc.MinConns = cfg.MinConns
	pc.MaxConnLifetime = cfg.MaxConnLifetime
	pc.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
