// Package trending builds the public trending page. The store supplies a
// bounded candidate shortlist per window; the pure scoring package owns the hot
// rank and tie-breaks. Results are cached briefly because every window reads
// the same small candidate set.
package trending

import (
	"context"
	"sync"
	"time"

	"grumble/internal/domain/pain"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/scoring"
)

// Page and cache bounds.
const (
	// Limit is the maximum number of ranked pains on the page.
	Limit = 50
	// BreakdownLimit is how many categories/countries each sidebar shows.
	BreakdownLimit = 8
	// cacheTTL keeps a window's result for one minute.
	cacheTTL = time.Minute
)

// Store is the read persistence the service needs.
type Store interface {
	ListTrendingWindow(ctx context.Context, windowStart time.Time, limit int) ([]pain.Trending, error)
	ListTrendingCategoryCounts(ctx context.Context, windowStart time.Time, limit int) ([]pain.Count, error)
	ListTrendingCountryCounts(ctx context.Context, windowStart time.Time, limit int) ([]pain.Count, error)
}

// Clock supplies the current time for windows and cache expiry.
type Clock interface {
	Now() time.Time
}

// Category is one category link in the sidebar or browse grid.
type Category struct {
	Slug  string
	Name  string
	Count int
}

// Country is one country link in the sidebar or browse grid.
type Country struct {
	Code  string
	Name  string
	Count int
}

// Page is one rendered trending window.
type Page struct {
	Window     scoring.Window
	Pains      []pain.Pain
	Categories []Category
	Countries  []Country
}

type cacheEntry struct {
	loaded time.Time
	page   Page
}

// Service reads and caches trending pages.
type Service struct {
	store Store
	clock Clock

	mu    sync.RWMutex
	cache map[scoring.Window]cacheEntry
}

// New builds the trending service.
func New(store Store, clock Clock) *Service {
	return &Service{store: store, clock: clock, cache: make(map[scoring.Window]cacheEntry)}
}

// Get returns the ranked page for w. Unknown windows fall back to the week.
func (s *Service) Get(ctx context.Context, w scoring.Window) (Page, error) {
	if !w.Valid() {
		w = scoring.WindowWeek
	}
	now := s.clock.Now()

	s.mu.RLock()
	entry, ok := s.cache[w]
	s.mu.RUnlock()
	if ok && now.Sub(entry.loaded) < cacheTTL {
		return clonePage(entry.page), nil
	}

	start := w.Start(now)
	candidates, err := s.store.ListTrendingWindow(ctx, start, Limit)
	if err != nil {
		return Page{}, err
	}
	items := make([]scoring.TrendingItem, 0, len(candidates))
	byID := make(map[string]pain.Pain, len(candidates))
	for _, c := range candidates {
		id := c.Pain.ID.String()
		items = append(items, scoring.TrendingItem{
			ID:        id,
			Votes:     c.Votes7d,
			Urgency:   c.Pain.Urgency,
			CreatedAt: c.Pain.CreatedAt,
		})
		byID[id] = c.Pain
	}
	ranked := scoring.RankTrending(items, now, w)
	pains := make([]pain.Pain, 0, len(ranked))
	for _, item := range ranked {
		if len(pains) == Limit {
			break
		}
		pains = append(pains, byID[item.ID])
	}

	catRows, err := s.store.ListTrendingCategoryCounts(ctx, start, BreakdownLimit)
	if err != nil {
		return Page{}, err
	}
	countryRows, err := s.store.ListTrendingCountryCounts(ctx, start, BreakdownLimit)
	if err != nil {
		return Page{}, err
	}

	page := Page{
		Window:     w,
		Pains:      pains,
		Categories: categories(catRows),
		Countries:  countries(countryRows),
	}
	s.mu.Lock()
	s.cache[w] = cacheEntry{loaded: now, page: page}
	s.mu.Unlock()
	return clonePage(page), nil
}

// Breakdown returns the category and country links for w without the ranked
// list. The saved page uses it for its all-time browse grid.
func (s *Service) Breakdown(ctx context.Context, w scoring.Window) ([]Category, []Country, error) {
	page, err := s.Get(ctx, w)
	if err != nil {
		return nil, nil, err
	}
	return append([]Category(nil), page.Categories...), append([]Country(nil), page.Countries...), nil
}

func categories(rows []pain.Count) []Category {
	out := make([]Category, 0, len(rows))
	for _, row := range rows {
		if row.Count < 1 {
			continue
		}
		c, ok := taxonomy.CategoryBySlug(row.Key)
		if !ok {
			continue
		}
		out = append(out, Category{Slug: c.Slug, Name: c.Name, Count: row.Count})
	}
	return out
}

func countries(rows []pain.Count) []Country {
	out := make([]Country, 0, len(rows))
	for _, row := range rows {
		if row.Count < 1 {
			continue
		}
		c, ok := taxonomy.CountryByCode(row.Key)
		if !ok {
			continue
		}
		out = append(out, Country{Code: c.Code, Name: c.Name, Count: row.Count})
	}
	return out
}

func clonePage(p Page) Page {
	p.Pains = append([]pain.Pain(nil), p.Pains...)
	p.Categories = append([]Category(nil), p.Categories...)
	p.Countries = append([]Country(nil), p.Countries...)
	return p
}
