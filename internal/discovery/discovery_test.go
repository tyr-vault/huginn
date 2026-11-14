package discovery

import (
	"testing"

	"github.com/rdvid/huginn/internal/models"
)

func TestValidate(t *testing.T) {
	if err := Validate(&models.SearchRequest{}); err == nil {
		t.Fatal("expected error")
	}
	req := &models.SearchRequest{Query: "dentists"}
	if err := Validate(req); err != nil {
		t.Fatal(err)
	}
	ApplyDefaults(req, 5, 2, 100, "pt")
	if req.ExtractionDepth != 5 || req.Concurrency != 2 || req.Lang != "pt" || req.MaxResults != 100 {
		t.Fatalf("defaults not applied: %+v", req)
	}
}
