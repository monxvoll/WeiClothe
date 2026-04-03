package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	// TODO: replace in production with go get github.com/aws/aws-advanced-go-wrapper/pgx-driver@latest
)

type Connection struct {
	db *pgxpool.Pool
}

func NewConnection(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)

	if err != nil {
		return nil, err
	}

	config.MaxConns = 4                         // TODO: for production, set to 25-50
	config.MinConns = 2                         // TODO: for production, set to 10-20
	config.MaxConnLifetime = 10 * time.Minute   // TODO: for production, set to 10-20 minutes
	config.MaxConnIdleTime = 5 * time.Minute    // TODO: for production, set to 5-10 minutes
	config.HealthCheckPeriod = 30 * time.Second // TODO: for production, set to 10-20 seconds

	db, err := pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
