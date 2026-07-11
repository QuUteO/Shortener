-- +goose Up
-- Таблица для хранения ссылок
CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,
    long_url TEXT NOT NULL,
    short_code VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Таблица для аналитики переходов
CREATE TABLE IF NOT EXISTS clicks (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(255) NOT NULL,
    clicked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    user_agent TEXT NOT NULL,
    device_type VARCHAR(50) NOT NULL,
    browser VARCHAR(100) NOT NULL,
    os VARCHAR(100) NOT NULL,
    referrer TEXT NOT NULL,

    CONSTRAINT fk_url FOREIGN KEY (short_code) REFERENCES urls(short_code) ON DELETE CASCADE
);

-- 1. Чтобы мгновенно находить клики по конкретной ссылке при сборке аналитики
CREATE INDEX IF NOT EXISTS idx_clicks_short_code ON clicks(short_code);
-- 2. Составной индекс (код + дата) — очень ускорит группировку по дням/месяцам
CREATE INDEX IF NOT EXISTS idx_clicks_code_date ON clicks(short_code, clicked_at);

-- +goose Down
DROP TABLE IF EXISTS clicks;
DROP TABLE IF EXISTS urls;
