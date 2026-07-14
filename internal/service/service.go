package service

import (
	"Shortener/internal/models"
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ua-parser/uap-go/uaparser"
)

var (
	ErrExists      = errors.New("такой shortcode уже существует")
	ErrURLNotFound = errors.New("ссылка не найдена")

	uaParser = uaparser.NewFromSaved()
)

type URLRepository interface {
	CreateURL(ctx context.Context, url *models.URL) error
	GetURLByCode(ctx context.Context, shortCode string) (*models.URL, error)
	IsCodeExists(ctx context.Context, shortCode string) (bool, error)

	SetURLCache(ctx context.Context, shortCode, url string) error
	GetURLCache(ctx context.Context, shortCode string) (string, error)

	SaveClick(ctx context.Context, click *models.Click) error
	GetClicksByCode(ctx context.Context, shortCode string) ([]models.Click, error)
}

type Service struct {
	repo URLRepository
}

func New(repository URLRepository) *Service {
	return &Service{
		repo: repository,
	}
}

func (s *Service) CreateURL(ctx context.Context, req models.ShortenRequest) (*models.ShortenResponse, error) {
	var shortCode string

	if req.CustomCode != "" {
		shortCode = req.CustomCode

		exists, err := s.repo.IsCodeExists(ctx, shortCode)
		if err != nil {
			return nil, err
		}

		if exists {
			return nil, ErrExists
		}
	} else {
		for {
			shortCode = s.generateRandomCode(6)

			exists, err := s.repo.IsCodeExists(ctx, shortCode)
			if err != nil {
				return nil, err
			}

			if !exists {
				break
			}
		}
	}

	urlModel := &models.URL{
		ID:        uuid.New(),
		LongURL:   req.LongURL,
		ShortCode: shortCode,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateURL(ctx, urlModel); err != nil {
		return nil, err
	}

	if err := s.repo.SetURLCache(ctx, shortCode, urlModel.LongURL); err != nil {
		log.Printf("не удалось сохранить URL в Redis: %v", err)
	}

	return &models.ShortenResponse{
		ShortCode: shortCode,
		ShortURL:  "http://localhost:8080/s/" + shortCode,
	}, nil
}

func (s *Service) GetOriginalURL(ctx context.Context, shortCode string, r *http.Request) (string, error) {

	longURL, err := s.repo.GetURLCache(ctx, shortCode)
	if err == nil && longURL != "" {
		go s.logClick(shortCode, r.Clone(context.Background()))
		return longURL, nil
	}

	urlModel, err := s.repo.GetURLByCode(ctx, shortCode)
	if err != nil {
		return "", ErrURLNotFound
	}

	if err := s.repo.SetURLCache(ctx, shortCode, urlModel.LongURL); err != nil {
		log.Printf("не удалось сохранить URL в Redis: %v", err)
	}

	go s.logClick(shortCode, r.Clone(context.Background()))

	return urlModel.LongURL, nil
}

func (s *Service) logClick(shortCode string, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userAgentRaw := r.Header.Get("User-Agent")
	referrer := r.Header.Get("Referer")

	client := uaParser.Parse(userAgentRaw)

	deviceType := "Desktop"

	uaLower := strings.ToLower(userAgentRaw)

	if strings.Contains(uaLower, "ipad") ||
		strings.Contains(uaLower, "tablet") {
		deviceType = "Tablet"
	} else if strings.Contains(uaLower, "iphone") ||
		strings.Contains(uaLower, "mobi") {
		deviceType = "Mobile"
	}

	click := &models.Click{
		ID:         uuid.New(),
		ShortCode:  shortCode,
		ClickedAt:  time.Now(),
		UserAgent:  userAgentRaw,
		DeviceType: deviceType,
		Browser:    client.UserAgent.Family,
		OS:         client.Os.Family,
		Referrer:   referrer,
	}

	if err := s.repo.SaveClick(ctx, click); err != nil {
		log.Printf("ошибка сохранения клика (%s): %v", shortCode, err)
	}
}

func (s *Service) GetAnalytics(ctx context.Context, shortCode string) (*models.AnalyticsResponse, error) {

	exists, err := s.repo.IsCodeExists(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrURLNotFound
	}

	clicks, err := s.repo.GetClicksByCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	response := &models.AnalyticsResponse{
		ShortCode:    shortCode,
		TotalClicks:  int64(len(clicks)),
		ByDay:        make(map[string]int),
		ByMonth:      make(map[string]int),
		ByDevice:     make(map[string]int),
		ByBrowser:    make(map[string]int),
		RecentClicks: []models.Click{},
	}

	for _, click := range clicks {

		day := click.ClickedAt.Format("2006-01-02")
		month := click.ClickedAt.Format("2006-01")

		response.ByDay[day]++
		response.ByMonth[month]++

		if click.DeviceType == "" {
			response.ByDevice["Unknown"]++
		} else {
			response.ByDevice[click.DeviceType]++
		}

		if click.Browser == "" {
			response.ByBrowser["Unknown"]++
		} else {
			response.ByBrowser[click.Browser]++
		}
	}

	limit := 20
	if len(clicks) < limit {
		limit = len(clicks)
	}

	response.RecentClicks = clicks[:limit]

	return response, nil
}

func (s *Service) generateRandomCode(length int) string {

	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	result := make([]byte, length)

	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			result[i] = alphabet[0]
			continue
		}
		result[i] = alphabet[n.Int64()]
	}

	return string(result)
}
