package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rdvid/huginn/internal/jobs"
	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/pipeline"
	"github.com/rdvid/huginn/internal/upstream/gmaps"
)

type fakeScraper struct {
	entries []*gmaps.Entry
	err     error
}

func (f *fakeScraper) Search(context.Context, models.SearchRequest) ([]*gmaps.Entry, error) {
	return f.entries, f.err
}

func (f *fakeScraper) Close() error { return nil }

func TestHealthAndSearchFlow(t *testing.T) {
	sc := &fakeScraper{entries: []*gmaps.Entry{{
		PlaceID: "x", Title: "Shop", Category: "Barber",
	}}}
	runner := &pipeline.Runner{Scraper: sc}
	mgr := jobs.NewManager(runner, 1)
	srv := &Server{Jobs: mgr}
	h := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("health status=%d", rr.Code)
	}

	body, _ := json.Marshal(models.SearchRequest{Query: "barbershops", Location: "SJC", MaxResults: 10})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("search status=%d body=%s", rr.Code, rr.Body.String())
	}

	var job models.Job
	if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := mgr.Get(job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == models.JobCompleted {
			results, err := mgr.Results(job.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != 1 || results[0].Name != "Shop" {
				t.Fatalf("results=%+v", results)
			}
			return
		}
		if got.Status == models.JobFailed {
			t.Fatalf("job failed: %s", got.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timeout waiting for job")
}
