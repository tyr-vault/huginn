package engineapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/rdvid/huginn/internal/config"
	"github.com/rdvid/huginn/internal/discovery"
	"github.com/rdvid/huginn/internal/jobs"
	"github.com/rdvid/huginn/internal/models"
)

// Server is the internal scrape-engine HTTP API.
type Server struct {
	Jobs     *jobs.Manager
	Log      *slog.Logger
	Defaults config.Config
}

// Handler returns routes for extract/jobs.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /v1/extract", s.handleExtract)
	mux.HandleFunc("GET /v1/jobs/{id}", s.handleGetJob)
	mux.HandleFunc("GET /v1/jobs/{id}/results", s.handleResults)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {
	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}
	discovery.ApplyDefaults(&req, s.Defaults.ExtractionDepth, s.Defaults.Concurrency, s.Defaults.RequestPaceMS, s.Defaults.Lang)
	if err := discovery.Validate(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Detach from request lifetime so the scrape outlives the HTTP call.
	job, err := s.Jobs.Submit(context.Background(), req)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"job_id": job.ID,
		"status": job.Status,
	})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	job, err := s.Jobs.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"job_id":       job.ID,
		"status":       job.Status,
		"error":        job.Error,
		"created_at":   job.CreatedAt,
		"updated_at":   job.UpdatedAt,
		"result_count": len(job.Results),
	})
}

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	results, err := s.Jobs.Results(id)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "job not found: "+id {
			status = http.StatusNotFound
		}
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
