// Package taxonomy holds the fixed lists of categories and countries a pain
// can be filed under. It is pure data and rules: no I/O, no other module
// imports (PRD FR-SUB-2, FR-SUB-3).
package taxonomy

// Category is a pain category. The zero value is invalid.
type Category struct {
	// Slug is the URL- and storage-safe identifier, for example "tech".
	Slug string
	// Name is the human-readable label, for example "Tech".
	Name string
}

// Valid reports whether c is one of the defined categories.
func (c Category) Valid() bool {
	for _, known := range allCategories {
		if known.Slug == c.Slug {
			return true
		}
	}
	return false
}

// Categories returns every category in display order. The returned slice is a
// copy and safe to modify.
func Categories() []Category {
	out := make([]Category, len(allCategories))
	copy(out, allCategories)
	return out
}

// CategoryBySlug returns the category with the given slug.
func CategoryBySlug(slug string) (Category, bool) {
	for _, c := range allCategories {
		if c.Slug == slug {
			return c, true
		}
	}
	return Category{}, false
}

// Country is an ISO 3166-1 alpha-2 territory, or the "Other" sentinel. The
// zero value is invalid.
type Country struct {
	// Code is the upper-case alpha-2 code, for example "US"; "XX" is Other.
	Code string
	// Name is the English short name, for example "United States".
	Name string
}

// CountryOther is the fallback for a location not in the ISO list.
var CountryOther = Country{Code: "XX", Name: "Other"}

// Valid reports whether c is a known ISO code or the Other sentinel.
func (c Country) Valid() bool {
	if c.Code == CountryOther.Code {
		return true
	}
	for _, known := range allCountries {
		if known.Code == c.Code {
			return true
		}
	}
	return false
}

// Countries returns every country sorted by English name, with Other last.
// The returned slice is a copy and safe to modify.
func Countries() []Country {
	out := make([]Country, 0, len(allCountries)+1)
	out = append(out, allCountries...)
	out = append(out, CountryOther)
	return out
}

// CountryByCode returns the country with the given alpha-2 code.
func CountryByCode(code string) (Country, bool) {
	if code == CountryOther.Code {
		return CountryOther, true
	}
	for _, c := range allCountries {
		if c.Code == code {
			return c, true
		}
	}
	return Country{}, false
}
