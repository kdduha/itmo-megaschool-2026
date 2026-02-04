package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kdduha/itmo-megaschool-2026/backend/internal/cache"
	"github.com/kdduha/itmo-megaschool-2026/backend/internal/config"
	"github.com/kdduha/itmo-megaschool-2026/backend/internal/handler"
	"github.com/kdduha/itmo-megaschool-2026/backend/internal/metrics"
	"github.com/kdduha/itmo-megaschool-2026/backend/internal/service"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "github.com/kdduha/itmo-megaschool-2026/backend/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger := log.Default()
	explainService := service.NewExplainService(
		logger,
		openai.NewClient(
			option.WithAPIKey(cfg.OpenAI.APIKey),
			option.WithBaseURL(cfg.OpenAI.BaseURL),
		), cfg.OpenAI)

	if cfg.CacheEnable {
		redisCache := cache.NewRedisCache(
			cfg.RedisConfig.Addr,
			cfg.RedisConfig.Password,
			cfg.RedisConfig.DB,
			cfg.RedisConfig.TTL,
		)
		explainService.SetCacheClient(redisCache)
		logger.Println("set redis as cache")
	}

	e := handler.NewExplainHandler(explainService)

	r := chi.NewRouter()
	r.Use([]func(http.Handler) http.Handler{
		middleware.Logger,
		middleware.Recoverer,
		middleware.Throttle(cfg.Server.ThrottleLimit),
		middleware.Timeout(cfg.Server.Timeout),
		metrics.Middleware,
	}...)

	r.Post("/explain", e.Explain)
	r.Post("/explain/stream", e.ExplainStream)
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	go func() {
		logger.Printf("server started :%s\n", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("listen error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatalf("server forced to shutdown: %v", err)
	}
	logger.Println("server stopped")
}
