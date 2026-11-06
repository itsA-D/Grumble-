package trending

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"grumble/internal/domain/pain"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/scoring"
)

var testNow = time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

type fakeStore struct {
	items      []pain.Trending
	categories []pain.Count
	countries  []pain.Count
	listCalls  int
	gotStart   time.Time
}

func (f *fakeStore) ListTrendingWindow(_ context.Context, start time.Time, _ int) ([]pain.Trending, error) {
	f.listCalls++
	f.gotStart = start
	return f.items, nil
}

func (f *fakeStore) ListTrendingCategoryCounts(context.Context, time.Time, int) ([]pain.Count, error) {
	return f.categories, nil
}

func (f *fakeStore) ListTrendingCountryCounts(context.Context, time.Time, int) ([]pain.Count, error) {
	return f.countries, nil
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func item(id byte, votes int, age time.Duration) pain.Trending {
	cat, _ := taxonomy.CategoryBySlug("tech")
	country, _ := taxonomy.CountryByCode("GB")
	return pain.Trending{
		Pain: pain.Pain{
			ID:        uuid.UUID{id},
			Slug:      "pain",
			Body:      "a pain body that is long enough",
			Category:  cat,
			Country:   country,
			Urgency:   80,
			CreatedAt: testNow.Add(-age),
		},
		Votes7d: votes,
	}
}

func TestGetFiltersAndRanksWindow(t *testing.T) {
	store := &fakeStore{items: []pain.Trending{
		item(1, 1, 8*24*time.Hour),
		item(2, 9, time.Hour),
		item(3, 5, 48*time.Hour),
	}}
	svc := New(store, fixedClock{now: testNow})

	page, err := svc.Get(context.Background(), scoring.WindowWeek)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(page.Pains) != 2 {
		t.Fatalf("len = %d, want 2 inside the week", len(page.Pains))
	}
	if page.Pains[0].ID != (uuid.UUID{2}) {
		t.Fatalf("first = %v, want the most voted", page.Pains[0].ID)
	}
	wantStart := testNow.AddDate(0, 0, -7)
	if !store.gotStart.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", store.gotStart, wantStart)
	}
}

func TestGetCachesPerWindow(t *testing.T) {
	store := &fakeStore{}
	svc := New(store, fixedClock{now: testNow})

	if _, err := svc.Get(context.Background(), scoring.WindowToday); err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if _, err := svc.Get(context.Background(), scoring.WindowToday); err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if store.listCalls != 1 {
		t.Fatalf("store calls = %d, want one cached read", store.listCalls)
	}
}

func TestGetInvalidWindowFallsBack(t *testing.T) {
	svc := New(&fakeStore{}, fixedClock{now: testNow})
	page, err := svc.Get(context.Background(), scoring.Window("nope"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if page.Window != scoring.WindowWeek {
		t.Fatalf("window = %q, want week", page.Window)
	}
}
