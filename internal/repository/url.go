package repository

import (
	"Shortener/internal/models"
	"context"

	"github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

type Repository struct {
	conn     *pgxdriver.Postgres
	redis    *redis.Client
	strategy *retry.Strategy
}

func NewRepository(conn *pgxdriver.Postgres, redis *redis.Client, strategy *retry.Strategy) *Repository {
	return &Repository{
		conn:     conn,
		redis:    redis,
		strategy: strategy,
	}
}

func (r *Repository) CreateURL(ctx context.Context, url *models.URL) error {
	query := 'INSERT INTO urls (id, long_url, short_url, created_at) VALUES ($1, $2, $3, $4)'

	_, err := r.conn.Exec(ctx, query, url.ID, url.LongURL, url.ShortURL, url.CreatedAt)
	if err != nil {

	}
}
