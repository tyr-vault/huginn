package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rdvid/huginn/internal/config"
	"github.com/rdvid/huginn/internal/engineapi"
	"github.com/rdvid/huginn/internal/jobs"
	"github.com/rdvid/huginn/internal/pipeline"
	"github.com/rdvid/huginn/internal/scraper"
)

func main() {
	cfg := config.LoadFromEnv()
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	addr := envOr("SCRAPE_ENGINE_ADDR", ":8090")

	sc := scraper.New(scraper.Options{
		Concurrency:   cfg.Concurrency,
		Debug:         cfg.Debug,
		Proxies:       cfg.Proxies,
		ExitIdleAfter: cfg.ExitIdleAfter,
	})
	defer func() { _ = sc.Close() }()

	runner := &pipeline.Runner{Scraper: sc, Log: log}
	mgr := jobs.NewManager(runner, 1)
	srv := &engineapi.Server{Jobs: mgr, Log: log, Defaults: cfg}

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("scrape-engine listening", "addr", addr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Error("scrape-engine failed", "error", err)
			os.Exit(1)
		}
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
