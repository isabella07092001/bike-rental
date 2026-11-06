package domain

import "testing"

func TestCategory(t *testing.T) {
	tests := []struct {
		category  Category
		wantValid bool
		wantLabel string
		wantPrice int64
	}{
		{CategoryChildren, true, "Children's bike", 400},
		{CategoryCity, true, "City bike", 600},
		{CategoryMountain, true, "Mountain bike", 900},
		{CategoryRacing, true, "Racing bike", 1000},
		{"", false, "", 0},
		{"tandem", false, "", 0},
		{"City", false, "", 0}, // values are case-sensitive
	}
	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			if got := tt.category.Valid(); got != tt.wantValid {
				t.Errorf("Valid() = %v, want %v", got, tt.wantValid)
			}
			if got := tt.category.Label(); got != tt.wantLabel {
				t.Errorf("Label() = %q, want %q", got, tt.wantLabel)
			}
			if got := tt.category.HourlyPrice(); got != tt.wantPrice {
				t.Errorf("HourlyPrice() = %d, want %d", got, tt.wantPrice)
			}
		})
	}
}

func TestAllCategoriesAreValid(t *testing.T) {
	all := AllCategories()
	if len(all) != 4 {
		t.Fatalf("len(AllCategories()) = %d, want 4", len(all))
	}
	for _, c := range all {
		if !c.Valid() {
			t.Errorf("category %q from AllCategories() is not valid", c)
		}
	}
}
