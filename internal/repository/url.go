package repository

import (
	"Shortener/internal/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/redis"
)

type Repository struct {
	conn  *pgxdriver.Postgres
	redis *redis.Client
}

var (
	ErrCreateURL   = errors.New("ошибка сохранения url")
	ErrSetURL      = errors.New("ошибка сохранения url в redis")
	ErrGetURL      = errors.New("url не найден")
	ErrGetURLCache = errors.New("ошибка получения url из redis")

	ErrSaveClick = errors.New("ошибка сохранения click")
	ErrRows      = errors.New("ошибка получения строк")
	ErrScanClick = errors.New("ошибка чтения click")
)

func New(conn *pgxdriver.Postgres, redis *redis.Client) *Repository {
	return &Repository{
		conn:  conn,
		redis: redis,
	}
}

func (r *Repository) CreateURL(ctx context.Context, url *models.URL) error {

	query := `
		INSERT INTO urls (
			id,
			long_url,
			short_code,
			created_at
		)
		VALUES ($1,$2,$3,$4)
	`

	_, err := r.conn.Exec(
		ctx,
		query,
		url.ID,
		url.LongURL,
		url.ShortCode,
		url.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("%w: %v", ErrCreateURL, err)
	}

	return nil
}

func (r *Repository) GetURLByCode(ctx context.Context, shortCode string) (*models.URL, error) {

	query := `
		SELECT
			id,
			long_url,
			short_code,
			created_at
		FROM urls
		WHERE short_code = $1
	`

	var url models.URL

	err := r.conn.QueryRow(ctx, query, shortCode).Scan(
		&url.ID,
		&url.LongURL,
		&url.ShortCode,
		&url.CreatedAt,
	)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGetURL
		}

		return nil, fmt.Errorf("%w: %v", ErrGetURL, err)
	}

	return &url, nil
}

func (r *Repository) IsCodeExists(ctx context.Context, shortCode string) (bool, error) {

	query := `
		SELECT EXISTS(
			SELECT 1
			FROM urls
			WHERE short_code = $1
		)
	`

	var exists bool

	err := r.conn.QueryRow(ctx, query, shortCode).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *Repository) SetURLCache(ctx context.Context, shortCode, url string) error {

	if err := r.redis.Set(ctx, shortCode, url); err != nil {
		return fmt.Errorf("%w: %v", ErrSetURL, err)
	}

	return nil
}

func (r *Repository) GetURLCache(ctx context.Context, shortCode string) (string, error) {

	value, err := r.redis.Get(ctx, shortCode)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGetURLCache, err)
	}

	return value, nil
}

func (r *Repository) SaveClick(ctx context.Context, click *models.Click) error {

	query := `
		INSERT INTO clicks (
			id,
			short_code,
			clicked_at,
			user_agent,
			device_type,
			browser,
			os,
			referrer
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`

	_, err := r.conn.Exec(
		ctx,
		query,
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
		return fmt.Errorf("%w: %v", ErrSaveClick, err)
	}

	return nil
}

func (r *Repository) GetClicksByCode(ctx context.Context, shortCode string) ([]models.Click, error) {

	query := `
		SELECT
			id,
			short_code,
			clicked_at,
			user_agent,
			device_type,
			browser,
			os,
			referrer
		FROM clicks
		WHERE short_code = $1
		ORDER BY clicked_at DESC
	`

	rows, err := r.conn.Query(ctx, query, shortCode)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRows, err)
	}

	defer rows.Close()

	clicks := make([]models.Click, 0)

	for rows.Next() {

		var click models.Click

		err := rows.Scan(
			&click.ID,
			&click.ShortCode,
			&click.ClickedAt,
			&click.UserAgent,
			&click.DeviceType,
			&click.Browser,
			&click.OS,
			&click.Referrer,
		)

		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrScanClick, err)
		}

		clicks = append(clicks, click)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return clicks, nil
}
