package mapsvc

import (
	"context"

	"grumble/internal/domain"
	"grumble/internal/domain/mapdata"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/scoring"
	"grumble/internal/service/mapsvc/geometry"
)

// Store is the read persistence the map service needs.
type Store interface {
	// List returns all country stats for a category (empty = all categories).
	List(ctx context.Context, category string) ([]mapdata.StatRow, error)
	// ListTotals returns totals per country across all categories.
	ListTotals(ctx context.Context) ([]mapdata.StatRow, error)
	// CountCountriesWithPains counts distinct countries with at least one pain.
	CountCountriesWithPains(ctx context.Context) (int, error)
	// SumPainCount returns the total pain count across all countries.
	SumPainCount(ctx context.Context) (int, error)
	// SumVoteCount returns the total vote count across all countries.
	SumVoteCount(ctx context.Context) (int, error)
	// GetTopCountries returns the top N countries by pain count.
	GetTopCountries(ctx context.Context, limit int) ([]mapdata.CountryRow, error)
	// GetTopCountriesByCategory returns the top N countries for a category.
	GetTopCountriesByCategory(ctx context.Context, category string, limit int) ([]mapdata.CountryRow, error)
	// ListTopPainsByCountry returns top N published pains for a country.
	ListTopPainsByCountry(ctx context.Context, country, category string, limit int) ([]mapdata.PainRow, error)
}

// CountryStat is one country's stats for the current filter.
type CountryStat struct {
	Country    string
	PainCount  int
	VoteCount  int
	AvgUrgency int
	Bucket     scoring.Bucket
}

// CountryPathData holds pre-computed values for rendering a country path.
type CountryPathData struct {
	ISO2      string
	Name      string
	PathD     string
	Link      string
	Hx        string
	PainCount int
	Bucket    int
}

// DefaultSideState is the side panel state when no country is selected.
type DefaultSideState struct {
	TopCountries []TopCountry
}

// SelectedSideState is the side panel state when a country is selected.
type SelectedSideState struct {
	Country     string
	CountryCode string
	PainCount   int
	VoteCount   int
	AvgUrgency  int
	TopPains    []TopPain
	HasPains    bool
}

// SidePanelData is the data needed for the side panel.
type SidePanelData struct {
	// DefaultState is shown when no country is selected.
	DefaultState DefaultSideState
	// SelectedState is shown when a country is selected.
	SelectedState *SelectedSideState
	// Category is the active category filter, kept so CLEAR preserves it.
	Category string
}

// TopCountry is a country in the "TOP COUNTRIES" list.
type TopCountry struct {
	Code      string
	Name      string
	PainCount int
}

// TopPain is a pain in the side panel's pain list.
type TopPain struct {
	Slug         string
	Body         string
	Category     string
	CategorySlug string
	VoteCount    int
	Urgency      int
	Seeded       bool
}

// Page is the full map page data.
type Page struct {
	Title          string
	Description    string
	Countries      []CountryStat
	CountryPaths   []CountryPathData
	MaxPain        int
	Category       string
	Categories     []taxonomy.Category
	ActiveCategory string
	Region         taxonomy.Region
	Regions        []taxonomy.Region
	SidePanel      SidePanelData
	PainTotal      int
	CountryTotal   int
	VoteTotal      int
}

// Service reads map data.
type Service struct {
	store Store
}

// New builds the map service.
func New(store Store) *Service {
	return &Service{store: store}
}

// Query identifies one map page request.
type Query struct {
	Category string
	Region   taxonomy.Region
	Country  string
}

// Get resolves a map page.
func (s *Service) Get(ctx context.Context, q Query) (Page, error) {
	// Normalize category: empty or invalid -> all categories.
	category := ""
	if _, ok := taxonomy.CategoryBySlug(q.Category); ok {
		category = q.Category
	}

	// Normalize region.
	region := q.Region
	if region == "" {
		region = taxonomy.RegionWorld
	}

	// Get country stats for the current category filter.
	stats, err := s.store.List(ctx, category)
	if err != nil {
		return Page{}, err
	}

	// Convert to domain types and compute buckets. Keep a map so the geometry
	// pass below can look up each country's intensity in O(1).
	byCountry := make(map[string]CountryStat, len(stats))
	maxPain := 0
	for _, st := range stats {
		if st.PainCount > maxPain {
			maxPain = st.PainCount
		}
	}

	countries := make([]CountryStat, 0, len(stats))
	for _, st := range stats {
		bucket := scoring.BucketFor(st.PainCount, maxPain)
		cs := CountryStat{
			Country:    st.Country,
			PainCount:  st.PainCount,
			VoteCount:  st.VoteCount,
			AvgUrgency: st.AvgUrgency,
			Bucket:     bucket,
		}
		countries = append(countries, cs)
		byCountry[st.Country] = cs
	}

	// Build path data for every country we have geometry for, including those
	// with no data (bucket 0) so the world is always fully drawn.
	countryPaths := make([]CountryPathData, 0, len(geometry.AllCountries))
	for _, geo := range geometry.AllCountries {
		stat, ok := byCountry[geo.ISO2]
		if !ok {
			stat = CountryStat{Country: geo.ISO2, Bucket: scoring.BucketNone}
		}
		link := "/country/" + geo.ISO2
		hx := "/map/country/" + geo.ISO2
		if category != "" {
			link += "?category=" + category
			hx += "?category=" + category
		}
		countryPaths = append(countryPaths, CountryPathData{
			ISO2:      geo.ISO2,
			Name:      geo.Name,
			PathD:     geo.PathD,
			Link:      link,
			Hx:        hx,
			PainCount: stat.PainCount,
			Bucket:    int(stat.Bucket),
		})
	}

	// Totals for the stat strip.
	painTotal, _ := s.store.SumPainCount(ctx)
	countryTotal, _ := s.store.CountCountriesWithPains(ctx)
	voteTotal, _ := s.store.SumVoteCount(ctx)

	// Top countries for the default side panel, honouring the category filter.
	var topRows []mapdata.CountryRow
	if category == "" {
		topRows, err = s.store.GetTopCountries(ctx, 8)
	} else {
		topRows, err = s.store.GetTopCountriesByCategory(ctx, category, 8)
	}
	if err != nil {
		return Page{}, err
	}
	topCountries := make([]TopCountry, 0, len(topRows))
	for _, r := range topRows {
		c, _ := taxonomy.CountryByCode(r.Country)
		topCountries = append(topCountries, TopCountry{
			Code:      r.Country,
			Name:      c.Name,
			PainCount: r.PainCount,
		})
	}

	// Selected country state.
	var selected *SelectedSideState

	if q.Country != "" {
		c, ok := taxonomy.CountryByCode(q.Country)
		if !ok {
			return Page{}, domain.ErrNotFound
		}

		// Get the stats for this country+category.
		stat, err := s.getCountryStat(ctx, q.Country, category)
		if err != nil {
			return Page{}, err
		}

		// Get top pains for this country.
		painRows, _ := s.store.ListTopPainsByCountry(ctx, q.Country, category, 5)
		topPains := make([]TopPain, 0, len(painRows))
		for _, p := range painRows {
			cat, _ := taxonomy.CategoryBySlug(p.Category)
			topPains = append(topPains, TopPain{
				Slug:         p.Slug,
				Body:         p.Body,
				Category:     cat.Name,
				CategorySlug: cat.Slug,
				VoteCount:    p.VoteCount,
				Urgency:      p.Urgency,
				Seeded:       p.Seeded,
			})
		}

		selected = &SelectedSideState{
			Country:     c.Name,
			CountryCode: c.Code,
			PainCount:   stat.PainCount,
			VoteCount:   stat.VoteCount,
			AvgUrgency:  stat.AvgUrgency,
			TopPains:    topPains,
			HasPains:    len(topPains) > 0,
		}
	}

	sidePanel := SidePanelData{
		DefaultState:  DefaultSideState{TopCountries: topCountries},
		SelectedState: selected,
		Category:      category,
	}

	// Build title and description (never show zero counts).
	title := "World Pain Map"
	description := ""
	if painTotal > 0 && countryTotal > 0 {
		description = "Frustrations across the world, ranked by how much they hurt."
	} else {
		description = "No signal yet. Be the first to post a frustration."
	}

	return Page{
		Title:          title,
		Description:    description,
		Countries:      countries,
		CountryPaths:   countryPaths,
		MaxPain:        maxPain,
		Category:       category,
		Categories:     taxonomy.Categories(),
		ActiveCategory: category,
		Region:         region,
		Regions:        taxonomy.AllRegions(),
		SidePanel:      sidePanel,
		PainTotal:      painTotal,
		CountryTotal:   countryTotal,
		VoteTotal:      voteTotal,
	}, nil
}

// getCountryStat returns the stats for one country in the current filter. A
// country with no published pains in that filter is a legitimate selection, so
// it returns a zero stat instead of an error; the side panel then shows its
// empty state.
func (s *Service) getCountryStat(ctx context.Context, country, category string) (CountryStat, error) {
	stats, err := s.store.List(ctx, category)
	if err != nil {
		return CountryStat{}, err
	}
	for _, st := range stats {
		if st.Country == country {
			return CountryStat{
				Country:    st.Country,
				PainCount:  st.PainCount,
				VoteCount:  st.VoteCount,
				AvgUrgency: st.AvgUrgency,
			}, nil
		}
	}
	return CountryStat{Country: country}, nil
}
