package repository

import (
	"Shortener/internal/models"
	"context"
	"errors"

	"github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
)

type Repository struct {
	conn     *pgxdriver.Postgres
	redis    *redis.Client
	strategy *retry.Strategy
}

var (
	ErrCreateURL     = errors.New("ошибка сохранения url в базу данных")
	ErrSetUrl        = errors.New("ошибка сохранения url в базу данных")
	ErrSaveToURL     = errors.New("ошибка сохранения полей в структуру URL")
	ErrGetURLCache   = errors.New("ошибка получения значения из кеша")
	ErrSaveClick     = errors.New("ошибка сохранения click")
	ErrRowsGet       = errors.New("ошибка получения строк")
	ErrScanClick     = errors.New("ошибка сканирования строк")
	ErrIterationRows = errors.New("ошибка итерации по строкам")
)

func NewRepository(conn *pgxdriver.Postgres, redis *redis.Client, strategy *retry.Strategy) *Repository {
	return &Repository{
		conn:     conn,
		redis:    redis,
		strategy: strategy,
	}
}

func (r *Repository) CreateURL(ctx context.Context, url *models.URL) error {
	query := "INSERT INTO urls (id, long_url, short_code, created_at) VALUES ($1, $2, $3, $4)"

	_, err := r.conn.Exec(ctx, query, url.ID, url.LongURL, url.ShortCode, url.CreatedAt)
	if err != nil {
		return ErrCreateURL
	}

	return nil
}

func (r *Repository) GetURLByCode(ctx context.Context, shortCode string) (*models.URL, error) {
	query := "SELECT id, long_url, short_code, created_at FROM urls WHERE short_code = $1"

	var url models.URL
	err := r.conn.QueryRow(ctx, query, shortCode).Scan(&url.ID, &url.LongURL, &url.ShortCode, &url.CreatedAt)
	if err != nil {
		return nil, ErrSaveToURL
	}

	return &url, nil
}

func (r *Repository) IsCodeExists(ctx context.Context, shortCode string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)"

	var exists bool
	err := r.conn.QueryRow(ctx, query, shortCode).Scan(&exists)
	if err != nil {
		return false, ErrSaveToURL
	}

	return exists, nil
}

func (r *Repository) SetURLCache(ctx context.Context, url, shortCode string) error {
	err := r.redis.Set(ctx, shortCode, url)
	if err != nil {
		return ErrSetUrl
	}

	return nil
}

func (r *Repository) GetURLCache(ctx context.Context, shortCode string) (string, error) {
	value, err := r.redis.Get(ctx, shortCode)
	if err != nil {
		return "", ErrGetURLCache
	}

	return value, nil
}

func (r *Repository) SaveClick(ctx context.Context, click *models.Click) error {
	query := "INSERO INTO clicks (id, short_code, clicked_at, user_agent, device_type, browser, os, referrer) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"

	_, err := r.conn.Exec(ctx, query,
		click.ID,
		click.ShortCode,
		click.ClickedAt,
		click.UserAgent,
		click.DeviceType,
		click.Browser,
		click.OS,
		click.Referrer,
	)
	if err != nil {
		return ErrSaveClick
	}

	return nil
}

func (r *Repository) GetClicksByCode(ctx context.Context, shortCode string) ([]models.Click, error) {
	query := `SELECT id, short_code, clicked_at, user_agent, device_type, browser, os, referrer 
	          FROM clicks 
	          WHERE short_code = $1
	          ORDER BY clicked_at DESC`

	rows, err := r.conn.Query(ctx, query, shortCode)
	if err != nil {
		return nil, ErrRowsGet
	}
	defer rows.Close()

	var clicks []models.Click

	for rows.Next() {
		var c models.Click

		err := rows.Scan(
			&c.ID,
			&c.ShortCode,
			&c.ClickedAt,
			&c.UserAgent,
			&c.DeviceType,
			&c.Browser,
			&c.OS,
			&c.Referrer,
		)
		if err != nil {
			return nil, ErrScanClick
		}

		clicks = append(clicks, c)
	}

	if err := rows.Err(); err != nil {
		return nil, ErrIterationRows
	}

	return clicks, nil
}
