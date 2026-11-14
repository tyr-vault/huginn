package backendapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/rdvid/huginn/internal/auth"
	"github.com/rdvid/huginn/internal/discovery"
	"github.com/rdvid/huginn/internal/engineclient"
	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/store"
)

// Server is the public Huginn backend gateway.
type Server struct {
	Store  *store.Store
	Auth   *auth.Service
	Engine *engineclient.Client
	Log    *slog.Logger
}

// Handler mounts public API routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)

	mux.Handle("GET /api/v1/settings", s.protect(s.handleGetSettings))
	mux.Handle("PUT /api/v1/settings", s.protect(s.handlePutSettings))
	mux.Handle("POST /api/v1/sessions", s.protect(s.handleCreateSession))
	mux.Handle("GET /api/v1/sessions", s.protect(s.handleListSessions))
	mux.Handle("GET /api/v1/sessions/{id}", s.protect(s.handleGetSession))
	mux.Handle("GET /api/v1/sessions/{id}/leads", s.protect(s.handleSessionLeads))
	mux.Handle("GET /api/v1/sessions/{id}/export.csv", s.protect(s.handleExportCSV))
	mux.Handle("DELETE /api/v1/sessions/{id}", s.protect(s.handleDeleteSession))
	return mux
}

func (s *Server) protect(h http.HandlerFunc) http.Handler {
	return s.Auth.Middleware(h)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	token, err := s.Auth.Login(body.Username, body.Password)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.Store.GetSettings(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]any
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.Store.PutSettings(r.Context(), settings); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

type createSessionReq struct {
	Name   string               `json:"name"`
	Config models.SearchRequest `json:"config"`
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var body createSessionReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name == "" {
		body.Name = body.Config.Query
		if body.Name == "" {
			body.Name = "session"
		}
	}
	if err := discovery.Validate(&body.Config); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	sess, err := s.Store.CreateSession(r.Context(), body.Name, body.Config)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	go s.runSession(sess.ID, body.Config)
	writeJSON(w, http.StatusAccepted, sess)
}

func (s *Server) runSession(sessionID string, cfg models.SearchRequest) {
	ctx := context.Background()
	log := s.Log
	if log == nil {
		log = slog.Default()
	}

	_ = s.Store.UpdateSessionStatus(ctx, sessionID, models.JobRunning, "", "")

	jobID, err := s.Engine.StartExtract(ctx, cfg)
	if err != nil {
		_ = s.Store.UpdateSessionStatus(ctx, sessionID, models.JobFailed, err.Error(), "")
		log.Error("start extract failed", "session", sessionID, "error", err)
		return
	}
	_ = s.Store.UpdateSessionStatus(ctx, sessionID, models.JobRunning, "", jobID)

	job, err := s.Engine.WaitForCompletion(ctx, jobID, 3*time.Second)
	if err != nil {
		_ = s.Store.UpdateSessionStatus(ctx, sessionID, models.JobFailed, err.Error(), jobID)
		log.Error("wait extract failed", "session", sessionID, "error", err)
		return
	}
	if job.Status == models.JobFailed {
		_ = s.Store.UpdateSessionStatus(ctx, sessionID, models.JobFailed, job.Error, jobID)
		return
	}

	leads, err := s.Engine.Results(ctx, jobID)
	if err != nil {
		_ = s.Store.UpdateSessionStatus(ctx, sessionID, models.JobFailed, err.Error(), jobID)
		return
	}
	if err := s.Store.InsertLeads(ctx, sessionID, leads); err != nil {
		_ = s.Store.UpdateSessionStatus(ctx, sessionID, models.JobFailed, err.Error(), jobID)
		return
	}
	_ = s.Store.UpdateSessionStatus(ctx, sessionID, models.JobCompleted, "", jobID)
	log.Info("session completed", "session", sessionID, "leads", len(leads))
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListSessions(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []store.Session{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.Store.GetSession(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleSessionLeads(w http.ResponseWriter, r *http.Request) {
	leads, err := s.Store.ListLeads(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if leads == nil {
		leads = []models.Business{}
	}
	writeJSON(w, http.StatusOK, leads)
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	leads, err := s.Store.ListLeads(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="session-`+id+`.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"place_id", "name", "category", "address", "city", "state", "country", "postal_code",
		"latitude", "longitude", "phone", "website", "maps_url", "rating", "review_count", "business_status", "extracted_at",
	})
	for _, b := range leads {
		_ = cw.Write([]string{
			b.PlaceID, b.Name, b.Category, b.Address, b.City, b.State, b.Country, b.PostalCode,
			formatFloat(b.Latitude), formatFloat(b.Longitude), b.Phone, b.Website, b.MapsURL,
			formatFloat(b.Rating), strconv.Itoa(b.ReviewCount), b.Status, b.ExtractedAt.UTC().Format(time.RFC3339),
		})
	}
	cw.Flush()
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DeleteSession(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func formatFloat(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
