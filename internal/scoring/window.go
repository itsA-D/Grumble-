package scoring

import (
	"sort"
	"strings"
	"time"
)

// Window is the time range a trending list covers. The default is WindowWeek;
// unknown values fall back to it so a hand-edited URL still renders.
type Window string

// Trending windows.
const (
	WindowToday Window = "today"
	WindowWeek  Window = "week"
	WindowMonth Window = "month"
	WindowAll   Window = "all"
)

// WindowFrom parses a query value, defaulting to WindowWeek.
func WindowFrom(raw string) Window {
	switch w := Window(strings.ToLower(strings.TrimSpace(raw))); w {
	case WindowToday, WindowWeek, WindowMonth, WindowAll:
		return w
	default:
		return WindowWeek
	}
}

// Valid reports whether w is one of the defined windows.
func (w Window) Valid() bool {
	switch w {
	case WindowToday, WindowWeek, WindowMonth, WindowAll:
		return true
	default:
		return false
	}
}

// Label is the human-readable chip label.
func (w Window) Label() string {
	switch w {
	case WindowToday:
		return "Today"
	case WindowMonth:
		return "This month"
	case WindowAll:
		return "All time"
	default:
		return "This week"
	}
}

// Start returns the inclusive lower bound for the window. Windows are rolling
// except Today, which starts at UTC midnight. All time starts at the Unix epoch.
func (w Window) Start(now time.Time) time.Time {
	now = now.UTC()
	switch w {
	case WindowToday:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	case WindowMonth:
		return now.AddDate(0, 0, -30)
	case WindowAll:
		return time.Unix(0, 0).UTC()
	default:
		return now.AddDate(0, 0, -7)
	}
}

// Includes reports whether createdAt falls inside the window at now.
func (w Window) Includes(createdAt, now time.Time) bool {
	return !createdAt.Before(w.Start(now))
}

// TrendingItem is the pure input to the windowed hot rank.
type TrendingItem struct {
	// ID is a stable, unique string used only for deterministic tie-breaking.
	ID        string
	Votes     int
	Comments  int
	Urgency   int
	CreatedAt time.Time
}

// RankTrending filters items to the window and orders them by the hot score.
// Ties break by votes and then by descending ID so repeated requests are stable.
func RankTrending(items []TrendingItem, now time.Time, w Window) []TrendingItem {
	if !w.Valid() {
		w = WindowWeek
	}
	start := w.Start(now)
	out := make([]TrendingItem, 0, len(items))
	for _, item := range items {
		if !item.CreatedAt.Before(start) {
			out = append(out, item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		hi := Hot(out[i].Votes, out[i].Comments, out[i].Urgency, now.Sub(out[i].CreatedAt))
		hj := Hot(out[j].Votes, out[j].Comments, out[j].Urgency, now.Sub(out[j].CreatedAt))
		if hi != hj {
			return hi > hj
		}
		if out[i].Votes != out[j].Votes {
			return out[i].Votes > out[j].Votes
		}
		return out[i].ID > out[j].ID
	})
	return out
}

// HotFactors exposes the constants the SQL shortlist passes to Postgres so the
// formula has one source in this package.
type HotFactors struct {
	CommentWeight  float64
	AgeOffset      float64
	AgeExponent    float64
	UrgencyDivisor float64
}

// Factors returns the current hot-rank constants.
func Factors() HotFactors {
	return HotFactors{
		CommentWeight:  commentWeight,
		AgeOffset:      ageOffset,
		AgeExponent:    ageExponent,
		UrgencyDivisor: urgencyDivisor,
	}
}
