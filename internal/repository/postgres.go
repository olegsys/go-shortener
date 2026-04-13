package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type PostgresStorage struct {
	conn *pgx.Conn
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &PostgresStorage{conn: conn}, nil
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.conn.Ping(ctx)
}

func (p *PostgresStorage) Close(ctx context.Context) error {
	return p.conn.Close(ctx)
}
