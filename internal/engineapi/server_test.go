package engineapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rdvid/huginn/internal/config"
	"github.com/rdvid/huginn/internal/jobs"
	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/pipeline"
	"github.com/rdvid/huginn/internal/upstream/gmaps"
)

type fakeScraper struct{}

func (fakeScraper) Search(context.Context, models.SearchRequest) ([]*gmaps.Entry, error) {
	return []*gmaps.Entry{{PlaceID: "p1", Title: "Shop", Category: "X"}}, nil
}
func (fakeScraper) Close() error { return nil }

func TestExtractFlow(t *testing.T) {
	mgr := jobs.NewManager(&pipeline.Runner{Scraper: fakeScraper{}}, 1)
	srv := &Server{Jobs: mgr, Defaults: config.Config{ExtractionDepth: 5, Concurrency: 1, Lang: "en"}}
	h := srv.Handler()

	body, _ := json.Marshal(models.SearchRequest{Query: "cafes"})
	req := httptest.NewRequest(http.MethodPost, "/v1/extract", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var start map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &start)
	jobID, _ := start["job_id"].(string)
	if jobID == "" {
		t.Fatal("missing job_id")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req = httptest.NewRequest(http.MethodGet, "/v1/jobs/"+jobID+"/results", nil)
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusOK {
			var leads []models.Business
			_ = json.Unmarshal(rr.Body.Bytes(), &leads)
			if len(leads) != 1 {
				t.Fatalf("leads=%v", leads)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timeout")
}
