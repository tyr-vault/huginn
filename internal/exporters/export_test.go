package exporters

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rdvid/huginn/internal/models"
)

func sample() []models.Business {
	return []models.Business{{
		PlaceID: "p1", Name: "Shop", Category: "Barber",
		City: "SJC", Latitude: -23.1, Longitude: -45.9,
		Phone: "+5511", Website: "https://example.com",
		ExtractedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}}
}

func TestExportJSONAndCSV(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "out.json")
	csvPath := filepath.Join(dir, "out.csv")

	if err := Export(sample(), FormatJSON, jsonPath); err != nil {
		t.Fatal(err)
	}
	if err := Export(sample(), FormatCSV, csvPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []models.Business
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded[0].PlaceID != "p1" {
		t.Fatalf("unexpected json: %+v", decoded)
	}

	csvData, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(csvData) == 0 {
		t.Fatal("empty csv")
	}
}

func TestExportSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leads.db")
	if err := Export(sample(), FormatSQLite, path); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM businesses`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("count=%d", n)
	}
}
