package handler

import (
	"Shortener/internal/models"
	"context"
	"net/http"
)

type URLService interface {
	CreateURL(ctx context.Context, req models.ShortenRequest) (*models.ShortenResponse, error)
	GetOriginalURL(ctx context.Context, shortCode string, r *http.Request) (string, error)
	GetAnalytics(ctx context.Context, shortCode string) (*models.AnalyticsResponse, error)
}
