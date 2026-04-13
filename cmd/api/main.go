package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novaku/go-elastic-search/config"
	esinternal "github.com/novaku/go-elastic-search/internal/elasticsearch"
	"github.com/novaku/go-elastic-search/internal/router"
	"github.com/novaku/go-elastic-search/internal/service"
	"go.uber.org/zap"
)

func main() {
	// ── Config ───────────────────────────────────────
	cfg := config.Load()

	logger := cfg.BuildLogger()
	defer logger.Sync() //nolint:errcheck

	// ── Elasticsearch ────────────────────────────────
	esClient, err := esinternal.New(&cfg.Elasticsearch, logger)
	if err != nil {
		logger.Fatal("connecting to elasticsearch", zap.Error(err))
	}

	// ── Services ─────────────────────────────────────
	queryBuilder := service.NewElasticsearchQueryBuilder()
	productRepo := service.NewElasticsearchProductRepository(esClient, logger, queryBuilder)
	if err := productRepo.EnsureIndex(context.Background()); err != nil {
		logger.Fatal("initialising product service", zap.Error(err))
	}
	productSvc := service.NewProductService(productRepo, logger)
	healthChecker := service.NewElasticsearchHealthChecker(esClient)

	// ── HTTP Router ─────────────────────────────────
	r := router.New(cfg, productSvc, healthChecker, logger)

	// ── HTTP Server ─────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      r,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("server starting",
			zap.String("addr", srv.Addr),
			zap.String("env", cfg.App.Env),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server…")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("forced shutdown", zap.Error(err))
	}
	logger.Info("server exited")
}
