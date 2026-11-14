package discovery

import (
	"fmt"
	"strings"

	"github.com/rdvid/huginn/internal/models"
)

// Validate checks a search request for required fields and bounds.
func Validate(req *models.SearchRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Query) == "" && strings.TrimSpace(req.Category) == "" {
		return fmt.Errorf("query or category is required")
	}
	if req.MaxResults < 0 {
		return fmt.Errorf("max_results must be >= 0")
	}
	if req.ExtractionDepth < 0 {
		return fmt.Errorf("extraction_depth must be >= 0")
	}
	if req.Zoom != 0 && (req.Zoom < 1 || req.Zoom > 21) {
		return fmt.Errorf("zoom must be between 1 and 21")
	}
	return nil
}

// ApplyDefaults fills empty request fields from defaults.
func ApplyDefaults(req *models.SearchRequest, depth, concurrency, paceMS int, lang string) {
	if req.ExtractionDepth <= 0 {
		req.ExtractionDepth = depth
	}
	if req.Concurrency <= 0 {
		req.Concurrency = concurrency
	}
	if req.RequestPaceMS <= 0 {
		req.RequestPaceMS = paceMS
	}
	if req.Lang == "" {
		req.Lang = lang
	}
	if req.MaxResults == 0 {
		req.MaxResults = 100
	}
}
