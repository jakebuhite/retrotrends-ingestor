package jobs

import (
	"testing"

	"github.com/jakebuhite/retrotrends-ingestor/internal/models"
)

func TestParseListingType(t *testing.T) {
	tests := []struct {
		input string
		want  models.ListingType
	}{
		{"AUCTION", models.ListingTypeAuction},
		{"BEST_OFFER", models.ListingTypeBestOffer},
		{"FIXED_PRICE", models.ListingTypeBuyItNow},
		{"", models.ListingTypeBuyItNow},
		{"UNKNOWN", models.ListingTypeBuyItNow},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseListingType(tt.input)
			if got != tt.want {
				t.Errorf("parseListingType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
