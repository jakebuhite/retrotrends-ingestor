package ebay

import (
	"strings"

	"github.com/jakebuhite/retrotrends-ingestor/internal/models"
)

// ParseCondition derives a listing condition from the eBay condition label and raw title.
// Title signals take precedence over the generic eBay condition field because eBay's
// condition vocabulary (New/Used) is too coarse for the sealed/cib/loose distinction.
func ParseCondition(ebayCondition, title string) models.Condition {
	upper := strings.ToUpper(title)

	if containsAny(upper, "FACTORY SEALED", "SEALED", "NEW SEALED", "BRAND NEW SEALED") {
		return models.ConditionSealed
	}

	if containsAny(upper, "DISC ONLY", "GAME ONLY", "CARTRIDGE ONLY", "NO MANUAL", "NO BOX", "LOOSE") {
		return models.ConditionLoose
	}

	if containsAny(upper, "CIB", "COMPLETE IN BOX", "COMPLETE", "W/ BOX", "WITH BOX", "WITH MANUAL") {
		return models.ConditionCIB
	}

	// Fall back to eBay's condition label
	switch strings.ToLower(strings.TrimSpace(ebayCondition)) {
	case "new":
		// "New" without a sealed title signal → assume complete-in-box
		return models.ConditionCIB
	case "used", "very good", "good", "acceptable", "for parts or not working":
		return models.ConditionUnknown
	}

	return models.ConditionUnknown
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
