package taxonomy

import "testing"

func TestCategoryCount(t *testing.T) {
	if got := len(Categories()); got != 22 {
		t.Fatalf("Categories() = %d, want 22", got)
	}
}

func TestCategoriesValidAndUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, c := range Categories() {
		if !c.Valid() {
			t.Errorf("category %q reports invalid", c.Slug)
		}
		if c.Name == "" {
			t.Errorf("category %q has no name", c.Slug)
		}
		if seen[c.Slug] {
			t.Errorf("duplicate category slug %q", c.Slug)
		}
		seen[c.Slug] = true
	}
	if (Category{Slug: "nope"}).Valid() {
		t.Error("unknown category should be invalid")
	}
	if _, ok := CategoryBySlug("tech"); !ok {
		t.Error("CategoryBySlug(tech) not found")
	}
}

func TestCountries(t *testing.T) {
	all := Countries()
	if len(all) < 200 {
		t.Fatalf("expected the ISO list plus Other, got %d", len(all))
	}
	if all[len(all)-1] != CountryOther {
		t.Error("Other should be last")
	}
	for _, c := range all {
		if !c.Valid() {
			t.Errorf("country %q reports invalid", c.Code)
		}
	}
	if _, ok := CountryByCode("US"); !ok {
		t.Error("US should be a known country")
	}
	if CountryOther.Valid() != true {
		t.Error("Other should be valid")
	}
	if (Country{Code: "ZZ"}).Valid() {
		t.Error("unknown country should be invalid")
	}
}
