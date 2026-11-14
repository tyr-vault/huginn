//go:build live

package scraper

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rdvid/huginn/internal/models"
)

// Opt-in live browser extraction test.
// Run: go test -tags=live ./internal/scraper -run Live -count=1
func TestLiveSearch(t *testing.T) {
	if os.Getenv("HUGINN_LIVE") != "1" {
		t.Skip("set HUGINN_LIVE=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	sc := New(Options{Concurrency: 1, ExitIdleAfter: time.Minute})
	defer sc.Close()

	entries, err := sc.Search(ctx, models.SearchRequest{
		Query:           "barbershop",
		Location:        "São José dos Campos, SP",
		MaxResults:      5,
		ExtractionDepth: 3,
		Lang:            "pt",
		Concurrency:     1,
		RequestPaceMS:   500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	t.Logf("got %d entries; first=%q", len(entries), entries[0].Title)
}
