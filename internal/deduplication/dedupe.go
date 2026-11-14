package deduplication

import (
	"fmt"
	"math"
	"strings"

	"github.com/rdvid/huginn/internal/models"
)

const geoEpsilon = 0.0005 // ~55m

// Deduplicate removes duplicate businesses.
// Preferred key: PlaceID. Fallback: normalized website, phone, or name+geo.
func Deduplicate(in []models.Business) []models.Business {
	if len(in) == 0 {
		return nil
	}

	out := make([]models.Business, 0, len(in))
	seenPlace := map[string]int{}
	seenFallback := map[string]int{}

	for _, b := range in {
		if b.PlaceID != "" {
			key := strings.ToLower(b.PlaceID)
			if idx, ok := seenPlace[key]; ok {
				out[idx] = merge(out[idx], b)
				continue
			}
			seenPlace[key] = len(out)
			out = append(out, b)
			continue
		}

		key := fallbackKey(b)
		if key == "" {
			out = append(out, b)
			continue
		}
		if idx, ok := seenFallback[key]; ok {
			out[idx] = merge(out[idx], b)
			continue
		}
		// Fuzzy geo match against existing name keys without place id.
		if matched := matchByNameGeo(out, b); matched >= 0 {
			out[matched] = merge(out[matched], b)
			continue
		}
		seenFallback[key] = len(out)
		out = append(out, b)
	}

	return out
}

func fallbackKey(b models.Business) string {
	if b.Website != "" {
		return "web:" + strings.ToLower(b.Website)
	}
	if b.Phone != "" {
		return "phone:" + b.Phone
	}
	name := normalizeName(b.Name)
	if name == "" {
		return ""
	}
	if b.Latitude != 0 || b.Longitude != 0 {
		return fmt.Sprintf("namegeo:%s:%.3f:%.3f", name, b.Latitude, b.Longitude)
	}
	addr := strings.ToLower(strings.TrimSpace(b.Address))
	if addr != "" {
		return "nameaddr:" + name + ":" + addr
	}
	return ""
}

func matchByNameGeo(existing []models.Business, b models.Business) int {
	name := normalizeName(b.Name)
	if name == "" || (b.Latitude == 0 && b.Longitude == 0) {
		return -1
	}
	for i, e := range existing {
		if e.PlaceID != "" {
			continue
		}
		if normalizeName(e.Name) != name {
			continue
		}
		if e.Latitude == 0 && e.Longitude == 0 {
			continue
		}
		if math.Abs(e.Latitude-b.Latitude) <= geoEpsilon && math.Abs(e.Longitude-b.Longitude) <= geoEpsilon {
			return i
		}
	}
	return -1
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func merge(a, b models.Business) models.Business {
	if a.PlaceID == "" {
		a.PlaceID = b.PlaceID
	}
	if a.Name == "" {
		a.Name = b.Name
	}
	if a.Category == "" {
		a.Category = b.Category
	}
	if a.Address == "" {
		a.Address = b.Address
	}
	if a.City == "" {
		a.City = b.City
	}
	if a.State == "" {
		a.State = b.State
	}
	if a.Country == "" {
		a.Country = b.Country
	}
	if a.PostalCode == "" {
		a.PostalCode = b.PostalCode
	}
	if a.Phone == "" {
		a.Phone = b.Phone
	}
	if a.Website == "" {
		a.Website = b.Website
	}
	if a.MapsURL == "" {
		a.MapsURL = b.MapsURL
	}
	if a.Status == "" {
		a.Status = b.Status
	}
	if a.Latitude == 0 {
		a.Latitude = b.Latitude
	}
	if a.Longitude == 0 {
		a.Longitude = b.Longitude
	}
	if a.Rating == 0 {
		a.Rating = b.Rating
	}
	if a.ReviewCount == 0 {
		a.ReviewCount = b.ReviewCount
	}
	if len(a.OpenHours) == 0 {
		a.OpenHours = b.OpenHours
	}
	if len(a.Emails) == 0 {
		a.Emails = b.Emails
	}
	if len(a.Categories) == 0 {
		a.Categories = b.Categories
	}
	if a.Raw == nil {
		a.Raw = b.Raw
	}
	if b.ExtractedAt.After(a.ExtractedAt) {
		a.ExtractedAt = b.ExtractedAt
	}
	return a
}
