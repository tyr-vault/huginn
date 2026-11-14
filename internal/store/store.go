package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rdvid/huginn/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Store wraps Postgres persistence for sessions and leads.
type Store struct {
	db *sql.DB
}

// Open connects to Postgres.
func Open(databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Migrate applies SQL files from dir in lexical order.
func (s *Store) Migrate(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	for _, name := range files {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(string(body)); err != nil {
			return fmt.Errorf("migrate %s: %w", name, err)
		}
	}
	return nil
}

// Session is a BI operation row.
type Session struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Status      string              `json:"status"`
	Config      models.SearchRequest `json:"config"`
	Error       string              `json:"error,omitempty"`
	EngineJobID string              `json:"engine_job_id,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	StartedAt   *time.Time          `json:"started_at,omitempty"`
	FinishedAt  *time.Time          `json:"finished_at,omitempty"`
	LeadCount   int                 `json:"lead_count,omitempty"`
}

// CreateSession inserts a pending session.
func (s *Store) CreateSession(ctx context.Context, name string, cfg models.SearchRequest) (*Session, error) {
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO sessions (name, status, config)
		VALUES ($1, 'pending', $2)
		RETURNING id, name, status, config, COALESCE(error, ''), COALESCE(engine_job_id, ''),
		          created_at, updated_at, started_at, finished_at
	`, name, cfgJSON)
	return scanSession(row)
}

func (s *Store) GetSession(ctx context.Context, id string) (*Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.name, s.status, s.config, COALESCE(s.error, ''), COALESCE(s.engine_job_id, ''),
		       s.created_at, s.updated_at, s.started_at, s.finished_at,
		       (SELECT COUNT(*) FROM leads l WHERE l.session_id = s.id)
		FROM sessions s WHERE s.id = $1
	`, id)
	sess, err := scanSessionWithCount(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	return sess, err
}

func (s *Store) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.name, s.status, s.config, COALESCE(s.error, ''), COALESCE(s.engine_job_id, ''),
		       s.created_at, s.updated_at, s.started_at, s.finished_at,
		       (SELECT COUNT(*) FROM leads l WHERE l.session_id = s.id)
		FROM sessions s
		ORDER BY s.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		sess, err := scanSessionWithCountRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sess)
	}
	return out, rows.Err()
}

func (s *Store) UpdateSessionStatus(ctx context.Context, id, status, errMsg, engineJobID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET
			status = $2,
			error = NULLIF($3, ''),
			engine_job_id = COALESCE(NULLIF($4, ''), engine_job_id),
			updated_at = now(),
			started_at = CASE WHEN $2 = 'running' AND started_at IS NULL THEN now() ELSE started_at END,
			finished_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN now() ELSE finished_at END
		WHERE id = $1
	`, id, status, errMsg, engineJobID)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session not found")
	}
	return nil
}

// InsertLeads bulk-inserts normalized businesses for a session.
func (s *Store) InsertLeads(ctx context.Context, sessionID string, leads []models.Business) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM leads WHERE session_id = $1`, sessionID); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO leads (
			session_id, place_id, name, category, categories, address, city, state, country, postal_code,
			latitude, longitude, phone, website, maps_url, rating, review_count, open_hours, status, emails,
			extracted_at, raw
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22
		)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, b := range leads {
		cats, _ := json.Marshal(b.Categories)
		hours, _ := json.Marshal(b.OpenHours)
		emails, _ := json.Marshal(b.Emails)
		raw, _ := json.Marshal(b.Raw)
		var placeID any
		if b.PlaceID != "" {
			placeID = b.PlaceID
		}
		if _, err := stmt.ExecContext(ctx,
			sessionID, placeID, b.Name, nullStr(b.Category), cats, nullStr(b.Address), nullStr(b.City),
			nullStr(b.State), nullStr(b.Country), nullStr(b.PostalCode),
			nullFloat(b.Latitude), nullFloat(b.Longitude), nullStr(b.Phone), nullStr(b.Website), nullStr(b.MapsURL),
			nullFloat(b.Rating), b.ReviewCount, hours, nullStr(b.Status), emails,
			b.ExtractedAt.UTC(), raw,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListLeads returns leads for a session.
func (s *Store) ListLeads(ctx context.Context, sessionID string) ([]models.Business, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(place_id, ''), name, COALESCE(category, ''), categories, COALESCE(address, ''),
		       COALESCE(city, ''), COALESCE(state, ''), COALESCE(country, ''), COALESCE(postal_code, ''),
		       COALESCE(latitude, 0), COALESCE(longitude, 0), COALESCE(phone, ''), COALESCE(website, ''),
		       COALESCE(maps_url, ''), COALESCE(rating, 0), COALESCE(review_count, 0), open_hours,
		       COALESCE(status, ''), emails, COALESCE(extracted_at, now()), raw
		FROM leads WHERE session_id = $1 ORDER BY name
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Business
	for rows.Next() {
		var b models.Business
		var cats, hours, emails, raw []byte
		if err := rows.Scan(
			&b.PlaceID, &b.Name, &b.Category, &cats, &b.Address, &b.City, &b.State, &b.Country, &b.PostalCode,
			&b.Latitude, &b.Longitude, &b.Phone, &b.Website, &b.MapsURL, &b.Rating, &b.ReviewCount, &hours,
			&b.Status, &emails, &b.ExtractedAt, &raw,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(cats, &b.Categories)
		_ = json.Unmarshal(hours, &b.OpenHours)
		_ = json.Unmarshal(emails, &b.Emails)
		_ = json.Unmarshal(raw, &b.Raw)
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetSettings / PutSettings store JSON defaults under key "defaults".
func (s *Store) GetSettings(ctx context.Context) (map[string]any, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = 'defaults'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Store) PutSettings(ctx context.Context, settings map[string]any) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value) VALUES ('defaults', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, raw)
	return err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullFloat(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSession(row scannable) (*Session, error) {
	var sess Session
	var cfgRaw []byte
	var started, finished sql.NullTime
	if err := row.Scan(
		&sess.ID, &sess.Name, &sess.Status, &cfgRaw, &sess.Error, &sess.EngineJobID,
		&sess.CreatedAt, &sess.UpdatedAt, &started, &finished,
	); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(cfgRaw, &sess.Config)
	if started.Valid {
		t := started.Time.UTC()
		sess.StartedAt = &t
	}
	if finished.Valid {
		t := finished.Time.UTC()
		sess.FinishedAt = &t
	}
	return &sess, nil
}

func scanSessionWithCount(row scannable) (*Session, error) {
	var sess Session
	var cfgRaw []byte
	var started, finished sql.NullTime
	if err := row.Scan(
		&sess.ID, &sess.Name, &sess.Status, &cfgRaw, &sess.Error, &sess.EngineJobID,
		&sess.CreatedAt, &sess.UpdatedAt, &started, &finished, &sess.LeadCount,
	); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(cfgRaw, &sess.Config)
	if started.Valid {
		t := started.Time.UTC()
		sess.StartedAt = &t
	}
	if finished.Valid {
		t := finished.Time.UTC()
		sess.FinishedAt = &t
	}
	return &sess, nil
}

func scanSessionWithCountRows(rows *sql.Rows) (*Session, error) {
	return scanSessionWithCount(rows)
}
