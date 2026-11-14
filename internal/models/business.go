package models

import "time"

// SearchRequest describes a business discovery request.
type SearchRequest struct {
	Query           string  `json:"query"`
	Location        string  `json:"location,omitempty"`
	Category        string  `json:"category,omitempty"`
	MaxResults      int     `json:"max_results,omitempty"`
	ExtractionDepth int     `json:"extraction_depth,omitempty"`
	Lang            string  `json:"lang,omitempty"`
	Geo             string  `json:"geo,omitempty"` // "lat,lon"
	Zoom            int     `json:"zoom,omitempty"`
	ExtractEmail    bool    `json:"extract_email,omitempty"`
	ExtraReviews    bool    `json:"extra_reviews,omitempty"`
	Concurrency     int     `json:"concurrency,omitempty"`
	RequestPaceMS   int     `json:"request_pace_ms,omitempty"`
	GridBBox        string  `json:"grid_bbox,omitempty"`
	GridCellKm      float64 `json:"grid_cell_km,omitempty"`
}

// Business is Huginn's normalized lead record.
type Business struct {
	PlaceID     string              `json:"place_id,omitempty"`
	Name        string              `json:"name"`
	Category    string              `json:"category,omitempty"`
	Categories  []string            `json:"categories,omitempty"`
	Address     string              `json:"address,omitempty"`
	City        string              `json:"city,omitempty"`
	State       string              `json:"state,omitempty"`
	Country     string              `json:"country,omitempty"`
	PostalCode  string              `json:"postal_code,omitempty"`
	Latitude    float64             `json:"latitude,omitempty"`
	Longitude   float64             `json:"longitude,omitempty"`
	Phone       string              `json:"phone,omitempty"`
	Website     string              `json:"website,omitempty"`
	MapsURL     string              `json:"maps_url,omitempty"`
	Rating      float64             `json:"rating,omitempty"`
	ReviewCount int                 `json:"review_count,omitempty"`
	OpenHours   map[string][]string `json:"opening_hours,omitempty"`
	Status      string              `json:"business_status,omitempty"`
	Emails      []string            `json:"emails,omitempty"`
	ExtractedAt time.Time           `json:"extracted_at"`
	Raw         map[string]any      `json:"raw,omitempty"`
}

// JobStatus values for the async API.
const (
	JobPending   = "pending"
	JobRunning   = "running"
	JobCompleted = "completed"
	JobFailed    = "failed"
)

// Job is an asynchronous search job.
type Job struct {
	ID        string        `json:"id"`
	Status    string        `json:"status"`
	Request   SearchRequest `json:"request"`
	Error     string        `json:"error,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Results   []Business    `json:"-"`
}
