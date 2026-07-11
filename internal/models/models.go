package models

import (
	"time"

	"github.com/google/uuid"
)

type URL struct {
	ID        uuid.UUID `json:"id"`
	LongURL   string    `json:"long_url"`
	ShortURL  string    `json:"short_url"`
	CreatedAt time.Time `json:"created_at"`
}

type Click struct {
	ID        uint64    `json:"id"`
	ShortCode string    `json:"short_code"` // Индекс для связи с таблицей URL
	ClickedAt time.Time `json:"clicked_at"` // Точное время клика (нужно для группировки по дням/месяцам)

	// Данные, извлеченные из HTTP-заголовков
	UserAgent  string `json:"user_agent"`  // Сырая строка User-Agent
	DeviceType string `json:"device_type"` // Распарсенный тип: Mobile, Desktop, Tablet
	Browser    string `json:"browser"`     // Браузер: Chrome, Safari, Firefox
	OS         string `json:"os"`          // Операционная система: iOS, Android, Windows
	Referrer   string `json:"referrer"`    // Откуда пришли (заголовок Referer, например: "https://t.me/")
}

// ShortenRequest — то, что приходит в теле POST /shorten
type ShortenRequest struct {
	LongURL    string `json:"long_url"`    // Обязательное поле с длинной ссылкой
	CustomCode string `json:"custom_code"` // Опционально: если пользователь хочет свое имя ссылки
}

// ShortenResponse — то, что мы отдаем пользователю в ответ
type ShortenResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"` // Полная готовая ссылка (например: http://localhost:8080/s/abc123)
}

// AnalyticsResponse — структура ответа для GET /analytics/{short_url}
type AnalyticsResponse struct {
	ShortCode   string         `json:"short_code"`
	TotalClicks int64          `json:"total_clicks"` // Общее число переходов
	ByDay       map[string]int `json:"by_day"`       // Группировка по дням: {"2026-07-10": 15, "2026-07-11": 32}
	ByMonth     map[string]int `json:"by_month"`     // Группировка по месяцам: {"2026-07": 47}
	ByDevice    map[string]int `json:"by_device"`    // По девайсам: {"Desktop": 30, "Mobile": 17}
	ByBrowser   map[string]int `json:"by_browser"`   // По браузерам: {"Chrome": 25, "Safari": 22}

	// Список последних 10-20 сырых кликов, чтобы вывести красивую таблицу на UI
	RecentClicks []Click `json:"recent_clicks"`
}
