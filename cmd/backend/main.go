package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rdvid/huginn/internal/auth"
	"github.com/rdvid/huginn/internal/backendapi"
	"github.com/rdvid/huginn/internal/engineclient"
	"github.com/rdvid/huginn/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	addr := envOr("BACKEND_ADDR", ":8080")
	dbURL := envOr("DATABASE_URL", "postgres://huginn:huginn@localhost:5432/huginn?sslmode=disable")
	engineURL := envOr("SCRAPE_ENGINE_URL", "http://localhost:8090")
	migrationsDir := envOr("MIGRATIONS_DIR", "migrations")

	st, err := store.Open(dbURL)
	if err != nil {
		log.Error("database open failed", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := st.Migrate(migrationsDir); err != nil {
		log.Error("migrate failed", "error", err)
		os.Exit(1)
	}
	log.Info("migrations applied", "dir", migrationsDir)

	authSvc, err := auth.New(auth.Config{
		Username: envOr("ADMIN_USER", "admin"),
		Password: envOr("ADMIN_PASSWORD", "changeme"),
		Secret:   []byte(envOr("JWT_SECRET", "change-me-long-random-secret")),
		TTL:      24 * time.Hour,
	})
	if err != nil {
		log.Error("auth config invalid", "error", err)
		os.Exit(1)
	}

	srv := &backendapi.Server{
		Store:  st,
		Auth:   authSvc,
		Engine: engineclient.New(engineURL),
		Log:    log,
	}

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           withCORS(srv.Handler()),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("backend listening", "addr", addr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Error("backend failed", "error", err)
			os.Exit(1)
		}
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
