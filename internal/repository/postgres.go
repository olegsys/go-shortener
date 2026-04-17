package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/olegsys/go-shortener/internal/model"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("unable to apply migrations: %w", err)
	}

	db := stdlib.OpenDB(*cfg)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresStorage) Close(ctx context.Context) error {
	_ = ctx
	return p.db.Close()
}

func (p *PostgresStorage) Set(ctx context.Context, shortURL, longURL string) (string, bool, error) {
	query := `
		INSERT INTO short_urls (short_url, original_url)
		VALUES ($1, $2)
		ON CONFLICT (original_url) DO UPDATE
		SET original_url = EXCLUDED.original_url
		RETURNING short_url;
	`

	var storedShortURL string
	if err := p.db.QueryRowContext(ctx, query, shortURL, longURL).Scan(&storedShortURL); err != nil {
		return "", false, fmt.Errorf("insert short url: %w", err)
	}
	return storedShortURL, storedShortURL == shortURL, nil
}

func (p *PostgresStorage) SetBatch(ctx context.Context, pairs []model.URLPair) ([]model.URLPair, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO short_urls (short_url, original_url)
		VALUES ($1, $2)
		ON CONFLICT (original_url) DO UPDATE
		SET original_url = EXCLUDED.original_url
		RETURNING short_url;
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	results := make([]model.URLPair, len(pairs))
	for i, pair := range pairs {
		var actualShortID string
		if err := stmt.QueryRowContext(ctx, pair.ShortURL, pair.LongURL).Scan(&actualShortID); err != nil {
			return nil, err
		}
		results[i] = model.URLPair{
			ShortURL: actualShortID,
			LongURL:  pair.LongURL,
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return results, nil
}

func (p *PostgresStorage) Get(ctx context.Context, shortURL string) (string, bool, error) {
	query := `SELECT original_url FROM short_urls WHERE short_url = $1`

	var originalURL string
	err := p.db.QueryRowContext(ctx, query, shortURL).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("select original url: %w", err)
	}

	return originalURL, true, nil
}

func runMigrations(dsn string) error {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parse migration database config: %w", err)
	}

	migrationDB := stdlib.OpenDB(*cfg)
	defer migrationDB.Close()

	if err := migrationDB.Ping(); err != nil {
		return fmt.Errorf("ping migration database: %w", err)
	}

	driver, err := postgres.WithInstance(migrationDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create postgres migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
