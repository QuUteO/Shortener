package main

import (
	config "Shortener/internal"
	"context"
	"fmt"
	"os"
	"time"

	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/redis"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Init("./config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации конфига: %v\n", err)
		os.Exit(1)
	}

	logger, err := logger.InitLogger(
		logger.ZapEngine,
		"Shortener",
		cfg.Env.Env,
		logger.WithLevel(logger.InfoLevel),
		logger.WithRotation("logs/app.log", 100, 5, 30),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации логера: %v\n", err)
		os.Exit(1)
	}

	pg, err := pgxdriver.New(
		cfg.Postgres.DatabaseDSN,
		logger,
		pgxdriver.MaxPoolSize(50),
		pgxdriver.MaxConnAttempts(5),
		pgxdriver.BaseRetryDelay(100*time.Millisecond),
	)
	if err != nil {
		logger.Error("Ошибка инициализации базы данных", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	if err := pg.Ping(ctx); err != nil {
		logger.Error("Ошибка пинга базы данных", "error", err)
	}

	logger.Info("Успешный запуск и проверка базы данных")

	client := redis.New(cfg.Redis.Addr, cfg.Redis.Password, 0)
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		logger.Error("Ошибка пинга кеша", "error", err)
	}

	logger.Info("Успешный запуск и проверка кеша")

	// strategy := retry.Strategy{Attempts: cfg.Redis.Attempts, Delay: 5 * time.Second, Backoff: cfg.Redis.Backoff}

}
