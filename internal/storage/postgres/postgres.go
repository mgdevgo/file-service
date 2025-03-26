package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(ctx context.Context, URL string) (*pgxpool.Pool, error) {
	const op = "storage.postgres.New"

	pool, err := pgxpool.New(ctx, URL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pingctx, cancelping := context.WithTimeout(ctx, time.Second*2)
	defer cancelping()

	if err := pool.Ping(pingctx); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := applyMigrations(URL); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return pool, nil
}

func applyMigrations(URL string) error {
	const op = "storage.postgres.applyMigrations"
	m, err := migrate.New(
		"file://migrations",
		strings.Replace(URL, "postgres://", "pgx5://", 1),
	)
	if err != nil {
		return fmt.Errorf("%s: failed to create migrate instance: %w", op, err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("%s: failed to run migrations: %w", op, err)
		}
	}

	return nil
}
