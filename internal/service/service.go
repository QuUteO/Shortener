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
	ErrExists          = errors.New("такой shortcode существует")
	ErrGetLongURLCache = errors.New("ошибка получения длинной ссылки из кеша")
	ErrURLNotFound     = errors.New("ошибка ссылка не найдена")

	uaParser = uaparser.NewFromSaved()
)

type URLRepository interface {
	// Ссылки (Postgres)
	CreateURL(ctx context.Context, url *models.URL) error
	GetURLByCode(ctx context.Context, shortCode string) (*models.URL, error)
	IsCodeExists(ctx context.Context, shortCode string) (bool, error)

	// Кэш (Redis)
	SetURLCache(ctx context.Context, url, shortCode string) error
	GetURLCache(ctx context.Context, shortCode string) (string, error)

	// Аналитика (Postgres)
	SaveClick(ctx context.Context, click *models.Click) error
	GetClicksByCode(ctx context.Context, shortCode string) ([]models.Click, error)
}

type Service struct {
	repo URLRepository
}

func NewService(repository URLRepository) Service {
	return Service{repo: repository}
}

// CreateURL — логика создания новой короткой ссылки
func (s *Service) CreateURL(ctx context.Context, req models.ShortenRequest) (*models.ShortenResponse, error) {
	var shortCode string

	// 1. Проверяем, передан ли CustomCode
	if req.CustomCode != "" {
		shortCode = req.CustomCode

		// Дергаем репозиторий, чтобы проверить уникальность кастомного кода
		exists, err := s.repo.IsCodeExists(ctx, shortCode)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrExists
		}
	} else {
		// Если CustomCode не передан -> запускаем цикл генерации случайного кода
		for {
			shortCode = s.generateRandomCode(6) // Генерируем код из 6 символов
			// Проверяем его уникальность в БД
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
	_ = s.repo.SetURLCache(ctx, shortCode, urlModel.LongURL)

	response := &models.ShortenResponse{
		ShortCode: shortCode,
		ShortURL:  "http://localhost:8080/s/" + shortCode,
	}

	return response, nil
}

// GetOriginalURL — логика быстрого поиска ссылки для редиректа
func (s *Service) GetOriginalURL(ctx context.Context, shortCode string, r *http.Request) (string, error) {
	// 1. Ищем код в кэше Redis
	longURL, err := s.repo.GetURLCache(ctx, shortCode)
	if err == nil && longURL != "" {
		go s.logClick(shortCode, r)
		return longURL, nil
	}

	// 2. Если в кэше пусто — идем в базу Postgres
	urlModel, err := s.repo.GetURLByCode(ctx, shortCode)
	if err != nil {
		return "", ErrURLNotFound
	}

	// 3. Раз нашли в базе — пишем в кэш Redis, чтобы следующий переход был мгновенным
	_ = s.repo.SetURLCache(ctx, shortCode, urlModel.LongURL)

	// 4. Запускаем фоновую горутину для сбора аналитики клика
	// Передаем r.Clone(ctx) или копируем нужные заголовки, чтобы избежать race condition,
	// так как http.Request очищается после завершения HTTP-запроса хендлера.
	reqCopy := r.Clone(context.Background())
	go s.logClick(shortCode, reqCopy)

	return urlModel.LongURL, nil
}

// logClick — приватный метод для фонового разбора заголовков и сохранения статистики
func (s *Service) logClick(shortCode string, r *http.Request) {
	// Создаем контекст с таймаутом для операции записи в БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userAgentRaw := r.Header.Get("User-Agent")
	referrer := r.Header.Get("Referer")

	// Парсим сырой User-Agent с помощью библиотеки uap-go
	client := uaParser.Parse(userAgentRaw)

	// Определяем тип устройства
	deviceType := "Desktop"

	uaLower := strings.ToLower(userAgentRaw)
	if strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "tablet") {
		deviceType = "Tablet"
	} else if strings.Contains(uaLower, "mobi") || strings.Contains(uaLower, "iphone") {
		deviceType = "Mobile"
	}

	click := &models.Click{
		ShortCode:  shortCode,
		ClickedAt:  time.Now(),
		UserAgent:  userAgentRaw,
		DeviceType: deviceType,
		Browser:    client.UserAgent.Family,
		OS:         client.Os.Family,
		Referrer:   referrer,
	}

	// Сохраняем в Postgres через репозиторий
	if err := s.repo.SaveClick(ctx, click); err != nil {
		log.Printf("[ERROR] ошибка сохранения клика %s: %v", shortCode, err)
	}
}

// GetAnalytics — сбор, агрегация и группировка статистики по короткой ссылке
func (s *Service) GetAnalytics(ctx context.Context, shortCode string) (*models.AnalyticsResponse, error) {
	// 1. Проверяем, существует ли вообще такая ссылка в нашей системе
	exists, err := s.repo.IsCodeExists(ctx, shortCode)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrURLNotFound
	}

	// 2. Запрашиваем из репозитория все сырые клики для этого кода
	// Репозиторий должен вернуть их отсортированными по убыванию времени (ORDER BY clicked_at DESC)
	clicks, err := s.repo.GetClicksByCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	// 3. Инициализируем структуру ответа и аллоцируем мапы
	res := &models.AnalyticsResponse{
		ShortCode:    shortCode,
		TotalClicks:  int64(len(clicks)),
		ByDay:        make(map[string]int),
		ByMonth:      make(map[string]int),
		ByDevice:     make(map[string]int),
		ByBrowser:    make(map[string]int),
		RecentClicks: make([]models.Click, 0),
	}

	// 4. Запускаем один цикл по всем кликам для группировки данных
	for _, click := range clicks {
		// Группировка по дням: "2026-07-13"
		day := click.ClickedAt.Format("2006-01-02")
		res.ByDay[day]++

		// Группировка по месяцам: "2026-07"
		month := click.ClickedAt.Format("2006-01")
		res.ByMonth[month]++

		// Группировка по типу устройства (Mobile, Desktop, Tablet)
		if click.DeviceType != "" {
			res.ByDevice[click.DeviceType]++
		} else {
			res.ByDevice["Неизвестное устройство"]++
		}

		// Группировка по браузерам (Chrome, Safari, Firefox)
		if click.Browser != "" {
			res.ByBrowser[click.Browser]++
		} else {
			res.ByBrowser["Неизвестный браузер"]++
		}
	}

	// 5. Берем первые 20 кликов для истории (последние переходы)
	// Так как в репозитории мы сделали ORDER BY clicked_at DESC, первые элементы — самые свежие
	limit := 20
	if len(clicks) < limit {
		limit = len(clicks)
	}

	// Делаем срез (slice) от 0 до лимита и сохраняем в ответ
	res.RecentClicks = clicks[:limit]

	return res, nil
}

// Вспомогательный метод для генерации криптографически стойкой случайной строки (Base62)
func (s *Service) generateRandomCode(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		result[i] = alphabet[num.Int64()]
	}

	return string(result)
}
