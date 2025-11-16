package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/vanya-egorov/PullRequest-Manager/internal/config"
	"github.com/vanya-egorov/PullRequest-Manager/internal/handler"
	"github.com/vanya-egorov/PullRequest-Manager/internal/infrastructure/postgres"
	"github.com/vanya-egorov/PullRequest-Manager/internal/usecase/pullrequest"
	"github.com/vanya-egorov/PullRequest-Manager/internal/usecase/stats"
	"github.com/vanya-egorov/PullRequest-Manager/internal/usecase/team"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

func main() {
	cfg := config.Load()
	logger := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer pool.Close()

	if cfg.Migrate {
		migrationsPath := "db/migrations/postgresql"
		if err := postgres.RunMigrations(ctx, pool, migrationsPath); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
		logger.Info("migrations completed")
	}

	repo := postgres.NewPostgresRepository(pool, logger)
	teamUC := team.New(repo, repo, logger)
	pullRequestUC := pullrequest.New(repo, repo, logger)
	statsUC := stats.New(repo, logger)
	h := handler.New(teamUC, pullRequestUC, statsUC, cfg.AdminToken, cfg.UserToken, logger)

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: h.Router(),
	}

	go func() {
		logger.Info("server listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("shutting down server")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}
