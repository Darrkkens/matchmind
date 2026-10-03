// Package store persists upstream API responses in PostgreSQL.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `CREATE TABLE IF NOT EXISTS api_responses (
	cache_key  text PRIMARY KEY,
	body       jsonb NOT NULL,
	fetched_at timestamptz NOT NULL DEFAULT now(),
	expires_at timestamptz
);
CREATE INDEX IF NOT EXISTS api_responses_expires_at ON api_responses (expires_at);`

// PostgresCache implements football.ResponseCache. Rows survive restarts, so a
// finished match is requested from the upstream API only once.
type PostgresCache struct{ pool *pgxpool.Pool }

func OpenPostgres(ctx context.Context, url string) (*PostgresCache, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresCache{pool: pool}, nil
}

func (c *PostgresCache) Close() { c.pool.Close() }

func (c *PostgresCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var body []byte
	err := c.pool.QueryRow(ctx, `SELECT body FROM api_responses WHERE cache_key = $1 AND (expires_at IS NULL OR expires_at > now())`, key).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return body, true, nil
}

func (c *PostgresCache) Put(ctx context.Context, key string, body []byte, ttl time.Duration) error {
	var expires *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expires = &t
	}
	_, err := c.pool.Exec(ctx, `INSERT INTO api_responses (cache_key, body, fetched_at, expires_at) VALUES ($1, $2, now(), $3)
		ON CONFLICT (cache_key) DO UPDATE SET body = EXCLUDED.body, fetched_at = EXCLUDED.fetched_at, expires_at = EXCLUDED.expires_at`, key, body, expires)
	return err
}
