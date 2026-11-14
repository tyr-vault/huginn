package normalization

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/rdvid/huginn/internal/models"
	"github.com/rdvid/huginn/internal/upstream/gmaps"
)

var nonDigitPlus = regexp.MustCompile(`[^\d+]`)

// FromEntry maps an upstream Entry into a Huginn Business with normalization.
func FromEntry(e *gmaps.Entry, extractedAt time.Time) models.Business {
	if extractedAt.IsZero() {
		extractedAt = time.Now().UTC()
	}

	b := models.Business{
		PlaceID:     strings.TrimSpace(e.PlaceID),
		Name:        strings.TrimSpace(e.Title),
		Category:    strings.TrimSpace(e.Category),
		Categories:  cleanStringSlice(e.Categories),
		Address:     strings.TrimSpace(e.Address),
		City:        strings.TrimSpace(e.CompleteAddress.City),
		State:       strings.TrimSpace(e.CompleteAddress.State),
		Country:     strings.TrimSpace(e.CompleteAddress.Country),
		PostalCode:  strings.TrimSpace(e.CompleteAddress.PostalCode),
		Latitude:    e.Latitude,
		Longitude:   e.Longtitude,
		Phone:       NormalizePhone(e.Phone),
		Website:     NormalizeWebsite(e.WebSite),
		MapsURL:     strings.TrimSpace(e.Link),
		Rating:      e.ReviewRating,
		ReviewCount: e.ReviewCount,
		OpenHours:   e.OpenHours,
		Status:      strings.TrimSpace(e.Status),
		Emails:      cleanStringSlice(e.Emails),
		ExtractedAt: extractedAt,
		Raw:         rawMap(e),
	}

	if b.Address == "" && e.CompleteAddress.Street != "" {
		parts := []string{e.CompleteAddress.Street, e.CompleteAddress.City, e.CompleteAddress.State, e.CompleteAddress.PostalCode, e.CompleteAddress.Country}
		b.Address = joinNonEmpty(parts, ", ")
	}

	return b
}

// FromEntries converts and normalizes a slice of entries.
func FromEntries(entries []*gmaps.Entry) []models.Business {
	now := time.Now().UTC()
	out := make([]models.Business, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		out = append(out, FromEntry(e, now))
	}
	return out
}

// NormalizePhone keeps a leading + and digits only.
func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	cleaned := nonDigitPlus.ReplaceAllString(phone, "")
	// Collapse multiple pluses / ensure plus only at start.
	if strings.Count(cleaned, "+") > 0 {
		hasPlus := strings.HasPrefix(cleaned, "+")
		digits := strings.Map(func(r rune) rune {
			if unicode.IsDigit(r) {
				return r
			}
			return -1
		}, cleaned)
		if hasPlus {
			return "+" + digits
		}
		return digits
	}
	return cleaned
}

// NormalizeWebsite canonicalizes URLs and unwraps common redirect wrappers.
func NormalizeWebsite(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if u, err := url.Parse(raw); err == nil {
		if (u.Host == "www.google.com" || u.Host == "google.com") && u.Path == "/url" {
			if q := u.Query().Get("q"); q != "" {
				raw = q
			} else if q := u.Query().Get("url"); q != "" {
				raw = q
			}
		}
	}

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	if (u.Scheme == "http" && u.Port() == "80") || (u.Scheme == "https" && u.Port() == "443") {
		u.Host = u.Hostname()
	}
	// Drop trailing slash on path root only for cleaner identity matching.
	if u.Path == "/" {
		u.Path = ""
	}
	return u.String()
}

func cleanStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

func joinNonEmpty(parts []string, sep string) string {
	var cleaned []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, sep)
}

func rawMap(e *gmaps.Entry) map[string]any {
	b, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}
