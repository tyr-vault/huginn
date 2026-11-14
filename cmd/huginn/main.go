package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/urfave/cli/v3"

	"github.com/rdvid/huginn/internal/api"
	"github.com/rdvid/huginn/internal/config"
	"github.com/rdvid/huginn/internal/exporters"
	"github.com/rdvid/huginn/internal/jobs"
	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/pipeline"
	"github.com/rdvid/huginn/internal/scraper"
)

func main() {
	cfg := config.LoadFromEnv()
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	app := &cli.Command{
		Name:  "huginn",
		Usage: "Lightweight self-hosted Google Maps lead discovery engine",
		Commands: []*cli.Command{
			searchCommand(cfg, log),
			exportCommand(cfg),
			serveCommand(cfg, log),
			installBrowsersCommand(),
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, os.Args); err != nil {
		log.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func searchCommand(cfg config.Config, log *slog.Logger) *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "Discover businesses from Google Maps",
		ArgsUsage: `"query"`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "location", Usage: "Geographic location"},
			&cli.StringFlag{Name: "category", Usage: "Business category"},
			&cli.IntFlag{Name: "limit", Value: 100, Usage: "Maximum results"},
			&cli.IntFlag{Name: "depth", Value: cfg.ExtractionDepth, Usage: "Feed scroll depth"},
			&cli.StringFlag{Name: "lang", Value: cfg.Lang, Usage: "Maps language code"},
			&cli.StringFlag{Name: "geo", Usage: "lat,lon for map center"},
			&cli.IntFlag{Name: "zoom", Value: 14, Usage: "Map zoom when --geo is set"},
			&cli.IntFlag{Name: "concurrency", Value: cfg.Concurrency, Usage: "Worker concurrency"},
			&cli.IntFlag{Name: "pace-ms", Value: cfg.RequestPaceMS, Usage: "Delay before scrape start"},
			&cli.StringFlag{Name: "output", Value: cfg.DefaultOutput, Usage: "Output file path"},
			&cli.StringFlag{Name: "format", Value: cfg.DefaultFormat, Usage: "json|csv|sqlite"},
			&cli.BoolFlag{Name: "email", Usage: "Optional website email enrichment"},
			&cli.BoolFlag{Name: "debug", Value: cfg.Debug, Usage: "Headful browser"},
			&cli.StringFlag{Name: "grid-bbox", Usage: "minLat,minLon,maxLat,maxLon"},
			&cli.Float64Flag{Name: "grid-cell-km", Value: 2.5, Usage: "Grid cell size km"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			query := cmd.Args().First()
			if query == "" && cmd.String("category") == "" {
				return fmt.Errorf("query argument or --category is required")
			}

			req := models.SearchRequest{
				Query:           query,
				Location:        cmd.String("location"),
				Category:        cmd.String("category"),
				MaxResults:      cmd.Int("limit"),
				ExtractionDepth: cmd.Int("depth"),
				Lang:            cmd.String("lang"),
				Geo:             cmd.String("geo"),
				Zoom:            cmd.Int("zoom"),
				Concurrency:     cmd.Int("concurrency"),
				RequestPaceMS:   cmd.Int("pace-ms"),
				ExtractEmail:    cmd.Bool("email"),
				GridBBox:        cmd.String("grid-bbox"),
				GridCellKm:      cmd.Float64("grid-cell-km"),
			}

			sc := scraper.New(scraper.Options{
				Concurrency:   cfg.Concurrency,
				Debug:         cmd.Bool("debug"),
				Proxies:       cfg.Proxies,
				ExitIdleAfter: cfg.ExitIdleAfter,
			})
			defer func() { _ = sc.Close() }()

			runner := &pipeline.Runner{Scraper: sc, Log: log}
			results, err := runner.Run(ctx, req)
			if err != nil {
				return err
			}

			out := cmd.String("output")
			if err := os.MkdirAll(filepath.Dir(absOrDot(out)), 0o755); err != nil && filepath.Dir(out) != "." {
				// ignore when writing to cwd filename
				_ = err
			}
			if err := exporters.Export(results, cmd.String("format"), out); err != nil {
				return err
			}
			log.Info("exported", "count", len(results), "format", cmd.String("format"), "output", out)
			return nil
		},
	}
}

func exportCommand(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Re-export a previously saved JSON leads file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "input", Required: true, Usage: "Input JSON file from huginn search"},
			&cli.StringFlag{Name: "format", Value: cfg.DefaultFormat, Usage: "json|csv|sqlite"},
			&cli.StringFlag{Name: "output", Value: "leads.out", Usage: "Output path"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			data, err := os.ReadFile(cmd.String("input"))
			if err != nil {
				return err
			}
			var businesses []models.Business
			if err := json.Unmarshal(data, &businesses); err != nil {
				return fmt.Errorf("parse input: %w", err)
			}
			return exporters.Export(businesses, cmd.String("format"), cmd.String("output"))
		},
	}
}

func serveCommand(cfg config.Config, log *slog.Logger) *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start the REST API",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "addr", Value: cfg.HTTPAddr, Usage: "Listen address"},
			&cli.IntFlag{Name: "concurrency", Value: cfg.Concurrency, Usage: "Scrape concurrency"},
			&cli.IntFlag{Name: "workers", Value: 1, Usage: "Max concurrent search jobs"},
			&cli.BoolFlag{Name: "debug", Value: cfg.Debug, Usage: "Headful browser"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			sc := scraper.New(scraper.Options{
				Concurrency:   cmd.Int("concurrency"),
				Debug:         cmd.Bool("debug"),
				Proxies:       cfg.Proxies,
				ExitIdleAfter: cfg.ExitIdleAfter,
			})
			defer func() { _ = sc.Close() }()

			runner := &pipeline.Runner{Scraper: sc, Log: log}
			mgr := jobs.NewManager(runner, cmd.Int("workers"))
			srv := &api.Server{Jobs: mgr, Log: log}

			httpServer := &http.Server{
				Addr:              cmd.String("addr"),
				Handler:           srv.Handler(),
				ReadHeaderTimeout: 10 * time.Second,
			}

			errCh := make(chan error, 1)
			go func() {
				log.Info("api listening", "addr", cmd.String("addr"))
				errCh <- httpServer.ListenAndServe()
			}()

			select {
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				_ = httpServer.Shutdown(shutdownCtx)
				return nil
			case err := <-errCh:
				if err == http.ErrServerClosed {
					return nil
				}
				return err
			}
		},
	}
}

func installBrowsersCommand() *cli.Command {
	return &cli.Command{
		Name:  "install-browsers",
		Usage: "Install Playwright Chromium for Huginn",
		Action: func(context.Context, *cli.Command) error {
			return playwright.Install()
		},
	}
}

func absOrDot(path string) string {
	if path == "" || path == "-" || path == "stdout" {
		return "."
	}
	if filepath.Dir(path) == "." {
		return "./" + path
	}
	return path
}
