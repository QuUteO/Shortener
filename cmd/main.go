package main

import (
	config "Shortener/internal"
	"Shortener/internal/handler"
	"Shortener/internal/repository"
	"Shortener/internal/service"
	"context"
	"fmt"
	"os"
	"time"

	pgxdriver "github.com/wb-go/wbf/dbpg/pgx-driver"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/logger"
	"github.com/wb-go/wbf/redis"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Init("./config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.InitLogger(
		logger.ZapEngine,
		"Shortener",
		cfg.Env.Env,
		logger.WithLevel(logger.InfoLevel),
		logger.WithRotation("logs/app.log", 100, 5, 30),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}

	pg, err := pgxdriver.New(
		cfg.Postgres.DatabaseDSN,
		log,
		pgxdriver.MaxPoolSize(50),
		pgxdriver.MaxConnAttempts(5),
		pgxdriver.BaseRetryDelay(100*time.Millisecond),
	)
	if err != nil {
		log.Error("ошибка подключения к PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	if err := pg.Ping(ctx); err != nil {
		log.Error("PostgreSQL недоступен", "error", err)
		os.Exit(1)
	}

	log.Info("PostgreSQL подключен")

	cache := redis.New(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		0,
	)
	defer cache.Close()

	if err := cache.Ping(ctx); err != nil {
		log.Error("Redis недоступен", "error", err)
		os.Exit(1)
	}

	log.Info("Redis подключен")

	repo := repository.New(pg, cache)
	srv := service.New(repo)
	urlHandler := handler.New(srv)

	router := ginext.New("debug")

	router.Use(
		ginext.Logger(),
		ginext.Recovery(),
	)

	router.GET("/", func(c *ginext.Context) {
		c.File("web/index.html")
	})
	router.POST("/shorten", urlHandler.CreateURL)
	router.GET("/s/:short_code", urlHandler.GetOriginalURL)
	router.GET("/analytics/:short_code", urlHandler.GetAnalytics)

	log.Info("HTTP server started", "addr", cfg.HTTP.Addr)

	if err := router.Run(cfg.HTTP.Addr); err != nil {
		log.Error("ошибка запуска HTTP сервера", "error", err)
		os.Exit(1)
	}
}
