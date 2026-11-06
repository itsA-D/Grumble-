package scoring

import (
	"testing"
	"time"
)

func TestWindowFromDefaultsWeek(t *testing.T) {
	for _, raw := range []string{"", "nonsense", "TOMORROW"} {
		if got := WindowFrom(raw); got != WindowWeek {
			t.Fatalf("WindowFrom(%q) = %q, want week", raw, got)
		}
	}
	if WindowFrom("TODAY") != WindowToday {
		t.Fatal("WindowFrom must be case-insensitive")
	}
}

func TestWindowBoundaryIsInclusive(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	start := WindowWeek.Start(now)
	items := []TrendingItem{
		{ID: "old", Votes: 100, CreatedAt: start.Add(-time.Nanosecond)},
		{ID: "edge", Votes: 1, CreatedAt: start},
	}
	got := RankTrending(items, now, WindowWeek)
	if len(got) != 1 || got[0].ID != "edge" {
		t.Fatalf("ranked = %+v, want only the inclusive boundary item", got)
	}
}

func TestRankTrendingTieBreaksByVotesThenID(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	items := []TrendingItem{
		{ID: "a", Votes: 5, CreatedAt: now.Add(-time.Hour)},
		{ID: "b", Votes: 9, CreatedAt: now.Add(-time.Hour)},
		{ID: "c", Votes: 5, CreatedAt: now.Add(-time.Hour)},
	}
	got := RankTrending(items, now, WindowWeek)
	want := []string{"b", "c", "a"}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("order = %v, want %v", ids(got), want)
		}
	}
}

func TestRankTrendingEmpty(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	if got := RankTrending(nil, now, WindowToday); len(got) != 0 {
		t.Fatalf("empty rank = %+v, want empty", got)
	}
}

func TestRankTrendingInvalidWindowFallsBack(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	items := []TrendingItem{{ID: "recent", CreatedAt: now.Add(-time.Hour)}}
	if got := RankTrending(items, now, Window("bogus")); len(got) != 1 {
		t.Fatalf("invalid window must fall back to week, got %+v", got)
	}
}

func TestWindowStartUsesMidnightForToday(t *testing.T) {
	now := time.Date(2026, 3, 10, 23, 59, 0, 0, time.UTC)
	want := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	if got := WindowToday.Start(now); !got.Equal(want) {
		t.Fatalf("start = %v, want %v", got, want)
	}
}

func ids(items []TrendingItem) []string {
	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].ID
	}
	return out
}
