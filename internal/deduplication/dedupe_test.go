package deduplication

import (
	"testing"

	"github.com/rdvid/huginn/internal/models"
)

func TestDeduplicateByPlaceID(t *testing.T) {
	in := []models.Business{
		{PlaceID: "A", Name: "One", Phone: ""},
		{PlaceID: "A", Name: "One", Phone: "+5511"},
		{PlaceID: "B", Name: "Two"},
	}
	out := Deduplicate(in)
	if len(out) != 2 {
		t.Fatalf("len=%d want 2", len(out))
	}
	if out[0].Phone != "+5511" {
		t.Fatalf("expected merge to keep phone, got %+v", out[0])
	}
}

func TestDeduplicateFallbackWebsite(t *testing.T) {
	in := []models.Business{
		{Name: "Shop", Website: "https://a.example"},
		{Name: "Shop Dup", Website: "https://a.example", Phone: "123"},
	}
	out := Deduplicate(in)
	if len(out) != 1 {
		t.Fatalf("len=%d want 1", len(out))
	}
	if out[0].Phone != "123" {
		t.Fatalf("expected phone merge")
	}
}

func TestDeduplicateNameGeo(t *testing.T) {
	in := []models.Business{
		{Name: "Cafe Central", Latitude: -23.1800, Longitude: -45.8800},
		{Name: "Cafe Central", Latitude: -23.1801, Longitude: -45.8801, Phone: "999"},
	}
	out := Deduplicate(in)
	if len(out) != 1 {
		t.Fatalf("len=%d want 1", len(out))
	}
}
