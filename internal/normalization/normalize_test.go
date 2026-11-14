package normalization

import (
	"testing"
	"time"

	"github.com/rdvid/huginn/internal/upstream/gmaps"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"(12) 3456-7890", "1234567890"},
		{"+55 12 3456-7890", "+551234567890"},
		{"  +1 (555) 111-2222 ", "+15551112222"},
	}
	for _, tt := range tests {
		if got := NormalizePhone(tt.in); got != tt.want {
			t.Fatalf("NormalizePhone(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeWebsite(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"example.com", "https://example.com"},
		{"https://Example.COM/path/", "https://example.com/path/"},
		{"https://www.google.com/url?q=https%3A%2F%2Fshop.example.com%2F", "https://shop.example.com"},
	}
	for _, tt := range tests {
		if got := NormalizeWebsite(tt.in); got != tt.want {
			t.Fatalf("NormalizeWebsite(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestFromEntry(t *testing.T) {
	e := &gmaps.Entry{
		PlaceID:      "ChIJabc",
		Title:        " Test Shop ",
		Category:     "Barber shop",
		Address:      "1 Main St",
		Phone:        "+55 12 99999-0000",
		WebSite:      "example.com",
		Link:         "https://maps.google.com/?cid=1",
		ReviewRating: 4.5,
		ReviewCount:  10,
		Latitude:     -23.1,
		Longtitude:   -45.9,
		Status:       "Open",
		CompleteAddress: gmaps.Address{
			City: "SJC", State: "SP", Country: "BR", PostalCode: "12200",
		},
	}
	b := FromEntry(e, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if b.PlaceID != "ChIJabc" || b.Name != "Test Shop" {
		t.Fatalf("unexpected identity: %+v", b)
	}
	if b.Phone != "+5512999990000" {
		t.Fatalf("phone=%q", b.Phone)
	}
	if b.Website != "https://example.com" {
		t.Fatalf("website=%q", b.Website)
	}
	if b.City != "SJC" || b.State != "SP" || b.Country != "BR" {
		t.Fatalf("geo fields: %+v", b)
	}
	if b.Longitude != -45.9 {
		t.Fatalf("longitude=%v", b.Longitude)
	}
	if b.ExtractedAt.IsZero() || b.Raw == nil {
		t.Fatalf("expected timestamp and raw payload")
	}
}
