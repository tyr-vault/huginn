package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rdvid/huginn/internal/deduplication"
	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/normalization"
	"github.com/rdvid/huginn/internal/scraper"
)

// Runner executes SEARCH → EXTRACT → NORMALIZE → DEDUPLICATE.
type Runner struct {
	Scraper scraper.Scraper
	Log     *slog.Logger
}

// Run executes the full discovery pipeline.
func (r *Runner) Run(ctx context.Context, req models.SearchRequest) ([]models.Business, error) {
	if r.Scraper == nil {
		return nil, fmt.Errorf("scraper is required")
	}
	log := r.Log
	if log == nil {
		log = slog.Default()
	}

	log.Info("search started",
		"query", req.Query,
		"location", req.Location,
		"max_results", req.MaxResults,
		"depth", req.ExtractionDepth,
	)

	entries, err := r.Scraper.Search(ctx, req)
	if err != nil {
		log.Error("extraction failed", "error", err)
		return nil, err
	}
	log.Info("extracted entries", "count", len(entries))

	businesses := normalization.FromEntries(entries)
	businesses = deduplication.Deduplicate(businesses)

	if req.MaxResults > 0 && len(businesses) > req.MaxResults {
		businesses = businesses[:req.MaxResults]
	}

	log.Info("pipeline complete", "businesses", len(businesses))
	return businesses, nil
}
