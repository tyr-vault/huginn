package pipeline

import (
	"context"
	"testing"

	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/upstream/gmaps"
)

type fakeScraper struct {
	entries []*gmaps.Entry
}

func (f *fakeScraper) Search(context.Context, models.SearchRequest) ([]*gmaps.Entry, error) {
	return f.entries, nil
}

func (f *fakeScraper) Close() error { return nil }

func TestRunnerDedupes(t *testing.T) {
	r := &Runner{Scraper: &fakeScraper{entries: []*gmaps.Entry{
		{PlaceID: "1", Title: "A", Category: "X"},
		{PlaceID: "1", Title: "A", Category: "X", Phone: "1"},
	}}}
	out, err := r.Run(context.Background(), models.SearchRequest{Query: "a", MaxResults: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Phone != "1" {
		t.Fatalf("expected merged phone")
	}
}
