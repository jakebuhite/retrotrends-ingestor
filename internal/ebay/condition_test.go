package ebay

import (
	"testing"

	"github.com/jakebuhite/retrotrends-ingestor/internal/models"
)

func TestParseCondition(t *testing.T) {
	tests := []struct {
		name          string
		ebayCondition string
		title         string
		want          models.Condition
	}{
		{
			name:          "sealed keyword in title",
			ebayCondition: "New",
			title:         "Super Mario Sunshine GameCube SEALED",
			want:          models.ConditionSealed,
		},
		{
			name:          "factory sealed in title",
			ebayCondition: "New",
			title:         "Zelda Wind Waker FACTORY SEALED Nintendo GameCube",
			want:          models.ConditionSealed,
		},
		{
			name:          "new sealed in title overrides used condition",
			ebayCondition: "Used",
			title:         "Mario Kart Double Dash NEW SEALED GameCube",
			want:          models.ConditionSealed,
		},
		{
			name:          "sealed is case-insensitive",
			ebayCondition: "New",
			title:         "pikmin gamecube sealed",
			want:          models.ConditionSealed,
		},
		{
			name:          "disc only",
			ebayCondition: "Used",
			title:         "Super Smash Bros Melee DISC ONLY GameCube",
			want:          models.ConditionLoose,
		},
		{
			name:          "game only",
			ebayCondition: "Used",
			title:         "Metroid Prime GAME ONLY GameCube",
			want:          models.ConditionLoose,
		},
		{
			name:          "no manual",
			ebayCondition: "Used",
			title:         "Luigi's Mansion NO MANUAL GameCube",
			want:          models.ConditionLoose,
		},
		{
			name:          "no box",
			ebayCondition: "Used",
			title:         "F-Zero GX NO BOX GameCube",
			want:          models.ConditionLoose,
		},
		{
			name:          "loose keyword",
			ebayCondition: "Used",
			title:         "Pikmin LOOSE GameCube",
			want:          models.ConditionLoose,
		},
		{
			name:          "cib abbreviation",
			ebayCondition: "Used",
			title:         "Resident Evil 4 CIB GameCube",
			want:          models.ConditionCIB,
		},
		{
			name:          "complete in box",
			ebayCondition: "Used",
			title:         "Paper Mario The Thousand Year Door Complete in Box GameCube",
			want:          models.ConditionCIB,
		},
		{
			name:          "with box abbreviation",
			ebayCondition: "Used",
			title:         "Donkey Kong Jungle Beat w/ Box GameCube",
			want:          models.ConditionCIB,
		},
		{
			name:          "with manual",
			ebayCondition: "Used",
			title:         "Eternal Darkness WITH MANUAL GameCube",
			want:          models.ConditionCIB,
		},
		{
			name:          "new ebay condition with no title signal defaults to cib",
			ebayCondition: "New",
			title:         "Super Mario Sunshine GameCube",
			want:          models.ConditionCIB,
		},
		{
			name:          "used ebay condition with no title signal is unknown",
			ebayCondition: "Used",
			title:         "Super Mario Sunshine GameCube",
			want:          models.ConditionUnknown,
		},
		{
			name:          "very good ebay condition with no title signal is unknown",
			ebayCondition: "Very Good",
			title:         "Star Wars Rogue Leader GameCube",
			want:          models.ConditionUnknown,
		},
		{
			name:          "empty condition and plain title is unknown",
			ebayCondition: "",
			title:         "Super Mario Sunshine GameCube",
			want:          models.ConditionUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCondition(tt.ebayCondition, tt.title)
			if got != tt.want {
				t.Errorf("ParseCondition(%q, %q) = %q, want %q",
					tt.ebayCondition, tt.title, got, tt.want)
			}
		})
	}
}
