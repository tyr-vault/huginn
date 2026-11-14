package scraper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/scrapemateapp"

	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/upstream/deduper"
	"github.com/rdvid/huginn/internal/upstream/exiter"
	"github.com/rdvid/huginn/internal/upstream/gmaps"
	"github.com/rdvid/huginn/internal/upstream/grid"
)

// Scraper discovers businesses from Google Maps.
type Scraper interface {
	Search(ctx context.Context, req models.SearchRequest) ([]*gmaps.Entry, error)
	Close() error
}

// Options configures the scrapemate-backed adapter.
type Options struct {
	Concurrency   int
	Debug         bool
	Proxies       []string
	DisableReuse  bool
	ExitIdleAfter time.Duration
}

// MateScraper wraps scrapemate + upstream gmaps jobs.
type MateScraper struct {
	opts Options
}

// New creates a MateScraper. A scrapemate app is created per Search call
// so concurrent API jobs do not share a single browser lifecycle.
func New(opts Options) *MateScraper {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 1
	}
	if opts.ExitIdleAfter <= 0 {
		opts.ExitIdleAfter = 2 * time.Minute
	}
	return &MateScraper{opts: opts}
}

func (s *MateScraper) Close() error { return nil }

// Search runs a full Playwright Maps crawl for the request.
func (s *MateScraper) Search(ctx context.Context, req models.SearchRequest) ([]*gmaps.Entry, error) {
	if req.Query == "" && req.Category == "" {
		return nil, fmt.Errorf("query or category is required")
	}

	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = s.opts.Concurrency
	}

	depth := req.ExtractionDepth
	if depth <= 0 {
		depth = 10
	}

	lang := req.Lang
	if lang == "" {
		lang = "en"
	}

	zoom := req.Zoom
	if zoom <= 0 {
		zoom = 14
	}

	collector := newCollector(req.MaxResults)
	appOpts := []func(*scrapemateapp.Config) error{
		scrapemateapp.WithConcurrency(concurrency),
		scrapemateapp.WithExitOnInactivity(s.opts.ExitIdleAfter),
	}
	if len(s.opts.Proxies) > 0 {
		appOpts = append(appOpts, scrapemateapp.WithProxies(s.opts.Proxies))
	}
	if s.opts.Debug {
		appOpts = append(appOpts, scrapemateapp.WithJS(scrapemateapp.Headfull(), scrapemateapp.DisableImages()))
	} else {
		appOpts = append(appOpts, scrapemateapp.WithJS(scrapemateapp.DisableImages()))
	}
	if !s.opts.DisableReuse {
		appOpts = append(appOpts,
			scrapemateapp.WithPageReuseLimit(2),
			scrapemateapp.WithBrowserReuseLimit(200),
		)
	}

	cfg, err := scrapemateapp.NewConfig([]scrapemate.ResultWriter{collector}, appOpts...)
	if err != nil {
		return nil, fmt.Errorf("scrapemate config: %w", err)
	}

	app, err := scrapemateapp.NewScrapeMateApp(cfg)
	if err != nil {
		return nil, fmt.Errorf("scrapemate app: %w", err)
	}
	defer func() { _ = app.Close() }()

	dedup := deduper.New()
	exitMonitor := exiter.New()

	query := buildQuery(req)
	seedJobs, err := createSeedJobs(req, query, lang, depth, zoom, dedup, exitMonitor)
	if err != nil {
		return nil, err
	}
	if len(seedJobs) == 0 {
		return nil, fmt.Errorf("no seed jobs created")
	}

	exitMonitor.SetSeedCount(len(seedJobs))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	exitMonitor.SetCancelFunc(cancel)
	go exitMonitor.Run(runCtx)

	if req.RequestPaceMS > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(req.RequestPaceMS) * time.Millisecond):
		}
	}

	err = app.Start(runCtx, seedJobs...)
	entries := collector.Entries()

	// Context cancel from exiter after completion is expected.
	if err != nil && runCtx.Err() == nil && ctx.Err() == nil {
		if len(entries) == 0 {
			return nil, err
		}
	}

	if req.MaxResults > 0 && len(entries) > req.MaxResults {
		entries = entries[:req.MaxResults]
	}

	return entries, nil
}

func buildQuery(req models.SearchRequest) string {
	q := req.Query
	if q == "" {
		q = req.Category
	} else if req.Category != "" {
		q = req.Category + " " + q
	}
	if req.Location != "" {
		q = q + " in " + req.Location
	}
	return q
}

func createSeedJobs(
	req models.SearchRequest,
	query, lang string,
	depth, zoom int,
	dedup deduper.Deduper,
	exitMonitor exiter.Exiter,
) ([]scrapemate.IJob, error) {
	opts := []gmaps.GmapJobOptions{
		gmaps.WithDeduper(dedup),
		gmaps.WithExitMonitor(exitMonitor),
	}
	if req.ExtraReviews {
		opts = append(opts, gmaps.WithExtraReviews())
	}

	if req.GridBBox != "" {
		bbox, err := grid.ParseBoundingBox(req.GridBBox)
		if err != nil {
			return nil, fmt.Errorf("grid bbox: %w", err)
		}
		cellKm := req.GridCellKm
		if cellKm <= 0 {
			cellKm = 2.5
		}
		cells := grid.GenerateCells(bbox, cellKm)
		jobs := make([]scrapemate.IJob, 0, len(cells))
		for _, cell := range cells {
			jobs = append(jobs, gmaps.NewGmapJob("", lang, query, depth, req.ExtractEmail, cell.GeoCoordinates(), zoom, opts...))
		}
		return jobs, nil
	}

	geo := req.Geo
	jobZoom := zoom
	if geo == "" {
		jobZoom = 0 // NewGmapJob requires both geo and zoom, or neither
	}
	job := gmaps.NewGmapJob("", lang, query, depth, req.ExtractEmail, geo, jobZoom, opts...)
	return []scrapemate.IJob{job}, nil
}

type collector struct {
	mu         sync.Mutex
	entries    []*gmaps.Entry
	maxResults int
	done       bool
}

func newCollector(maxResults int) *collector {
	return &collector{maxResults: maxResults}
}

func (c *collector) Run(_ context.Context, in <-chan scrapemate.Result) error {
	for result := range in {
		c.ingest(result.Data)
	}
	return nil
}

func (c *collector) ingest(data any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done {
		return
	}
	for _, item := range asSlice(data) {
		switch v := item.(type) {
		case *gmaps.Entry:
			c.entries = append(c.entries, v)
		case gmaps.Entry:
			e := v
			c.entries = append(c.entries, &e)
		}
		if c.maxResults > 0 && len(c.entries) >= c.maxResults {
			c.done = true
			return
		}
	}
}

func (c *collector) Entries() []*gmaps.Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*gmaps.Entry, len(c.entries))
	copy(out, c.entries)
	return out
}

func asSlice(t any) []any {
	if s, ok := t.([]any); ok {
		return s
	}
	if s, ok := t.([]*gmaps.Entry); ok {
		out := make([]any, len(s))
		for i, e := range s {
			out[i] = e
		}
		return out
	}
	return []any{t}
}
