package exporters

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rdvid/huginn/internal/models"
	_ "modernc.org/sqlite"
)

// Format names.
const (
	FormatJSON   = "json"
	FormatCSV    = "csv"
	FormatSQLite = "sqlite"
)

// Export writes businesses to path in the given format.
func Export(businesses []models.Business, format, path string) error {
	switch strings.ToLower(format) {
	case FormatJSON, "jsonl":
		return writeJSON(businesses, path)
	case FormatCSV:
		return writeCSV(businesses, path)
	case FormatSQLite:
		return writeSQLite(businesses, path)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func writeJSON(businesses []models.Business, path string) error {
	w, err := openWriter(path)
	if err != nil {
		return err
	}
	defer w.Close()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(businesses)
}

func writeCSV(businesses []models.Business, path string) error {
	w, err := openWriter(path)
	if err != nil {
		return err
	}
	defer w.Close()

	cw := csv.NewWriter(w)
	header := []string{
		"place_id", "name", "category", "address", "city", "state", "country",
		"postal_code", "latitude", "longitude", "phone", "website", "maps_url",
		"rating", "review_count", "business_status", "extracted_at",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, b := range businesses {
		row := []string{
			b.PlaceID,
			b.Name,
			b.Category,
			b.Address,
			b.City,
			b.State,
			b.Country,
			b.PostalCode,
			formatFloat(b.Latitude),
			formatFloat(b.Longitude),
			b.Phone,
			b.Website,
			b.MapsURL,
			formatFloat(b.Rating),
			strconv.Itoa(b.ReviewCount),
			b.Status,
			b.ExtractedAt.UTC().Format(time.RFC3339),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeSQLite(businesses []models.Business, path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	schema := `
CREATE TABLE IF NOT EXISTS businesses (
  place_id TEXT,
  name TEXT,
  category TEXT,
  address TEXT,
  city TEXT,
  state TEXT,
  country TEXT,
  postal_code TEXT,
  latitude REAL,
  longitude REAL,
  phone TEXT,
  website TEXT,
  maps_url TEXT,
  rating REAL,
  review_count INTEGER,
  business_status TEXT,
  extracted_at TEXT,
  raw_json TEXT
);
DELETE FROM businesses;
`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO businesses (
		place_id, name, category, address, city, state, country, postal_code,
		latitude, longitude, phone, website, maps_url, rating, review_count,
		business_status, extracted_at, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, b := range businesses {
		raw, _ := json.Marshal(b.Raw)
		if _, err := stmt.Exec(
			b.PlaceID, b.Name, b.Category, b.Address, b.City, b.State, b.Country, b.PostalCode,
			b.Latitude, b.Longitude, b.Phone, b.Website, b.MapsURL, b.Rating, b.ReviewCount,
			b.Status, b.ExtractedAt.UTC().Format(time.RFC3339), string(raw),
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type writeCloser struct {
	io.Writer
	closeFn func() error
}

func (w writeCloser) Close() error {
	if w.closeFn != nil {
		return w.closeFn()
	}
	return nil
}

func openWriter(path string) (io.WriteCloser, error) {
	if path == "" || path == "-" || path == "stdout" {
		return writeCloser{Writer: os.Stdout}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func formatFloat(v float64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
