// Package feed assembles the three feed tabs: Latest, Most voted and Trending
// (PRD FR-FEED-1, FR-FEED-2). Latest and Most voted are ordered in SQL and page
// by an opaque keyset cursor; Trending is finished here with the pure hot rank
// so the formula lives only in scoring. A category chip and a body search
// narrow every tab to published content only.
package feed

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"grumble/internal/domain/pain"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/scoring"
)

// Page-size and search bounds.
const (
	DefaultLimit = 20
	MaxLimit     = 100
	// MaxSearchLen caps the search term so an oversized query never reaches the
	// database.
	MaxSearchLen = 100
	// candidateLimit bounds the recent set the hot rank sorts in Go. Phase 2
	// replaces this with a maintained read model.
	candidateLimit = 200
)

// farFuture is the first-page sentinel for the keyset comparisons: no published
// row is ever created beyond it, so every row sorts before it.
var farFuture = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

// Tab selects the ordering.
type Tab string

// Feed tabs (FR-FEED-1).
const (
	Latest    Tab = "latest"
	MostVoted Tab = "most_voted"
	Trending  Tab = "trending"
)

// Store is the read persistence the service needs.
type Store interface {
	ListFeed(ctx context.Context, f pain.FeedFilter) ([]pain.Pain, error)
	ListTrendingCandidates(ctx context.Context, category, search string, limit int) ([]pain.Trending, error)
	CountPublished(ctx context.Context) (int, error)
	CountCountries(ctx context.Context) (int, error)
	CountVotes(ctx context.Context) (int, error)
}

// Clock supplies the current time for the hot rank's age term.
type Clock interface {
	Now() time.Time
}

// Service reads feed pages.
type Service struct {
	store Store
	clock Clock
}

// New builds the feed service.
func New(store Store, clk Clock) *Service {
	return &Service{store: store, clock: clk}
}

// Query is one feed page request. Category and Search are optional filters;
// Cursor is the opaque value from the previous page (empty starts from the top).
type Query struct {
	Tab      Tab
	Category string
	Search   string
	Cursor   string
	Limit    int
}

// Stats are published-only totals for the stat strip.
type Stats struct {
	Pains     int
	Countries int
	Votes     int
}

// Page is one feed page and the cursor for the next one (FR-FEED-1).
type Page struct {
	Pains      []pain.Pain
	NextCursor string
	HasNext    bool
	Stats      Stats
}

// List returns one page of the requested tab.
func (s *Service) List(ctx context.Context, q Query) (Page, error) {
	limit := normalizeLimit(q.Limit)
	category := normalizeCategory(q.Category)
	search := normalizeSearch(q.Search)
	after := decodeCursor(q.Cursor, q.Tab)

	if q.Tab == Trending {
		page, err := s.listTrending(ctx, limit, category, search, after)
		if err != nil {
			return Page{}, err
		}
		return s.withStats(ctx, page)
	}

	rows, err := s.store.ListFeed(ctx, pain.FeedFilter{
		Order:    orderFor(q.Tab),
		Category: category,
		Search:   search,
		After:    after.feedCursor(),
		// One extra row tells us whether a further page exists.
		Limit: limit + 1,
	})
	if err != nil {
		return Page{}, err
	}
	return s.withStats(ctx, pageOf(rows, limit, q.Tab))
}

// withStats attaches the board totals the header stat strip renders.
func (s *Service) withStats(ctx context.Context, page Page) (Page, error) {
	pains, err := s.store.CountPublished(ctx)
	if err != nil {
		return Page{}, err
	}
	countries, err := s.store.CountCountries(ctx)
	if err != nil {
		return Page{}, err
	}
	votes, err := s.store.CountVotes(ctx)
	if err != nil {
		return Page{}, err
	}
	page.Stats = Stats{Pains: pains, Countries: countries, Votes: votes}
	return page, nil
}

// listTrending ranks the candidate window by the pure hot score, then applies
// the cursor over that computed ordering. The window is bounded (candidateLimit)
// so this stays cheap; Phase 2 makes the ranking keyset-native.
func (s *Service) listTrending(ctx context.Context, limit int, category, search string, after cursor) (Page, error) {
	candidates, err := s.store.ListTrendingCandidates(ctx, category, search, candidateLimit)
	if err != nil {
		return Page{}, err
	}
	now := s.clock.Now()
	sort.SliceStable(candidates, func(i, j int) bool {
		hi, hj := hot(candidates[i], now), hot(candidates[j], now)
		if hi != hj {
			return hi > hj
		}
		// Deterministic tie-break so the keyset cursor is stable.
		return candidates[i].Pain.ID.String() > candidates[j].Pain.ID.String()
	})

	start := 0
	if after.hasHot {
		for start < len(candidates) {
			h := hot(candidates[start], now)
			if h < after.hot || (h == after.hot && candidates[start].Pain.ID.String() < after.id.String()) {
				break
			}
			start++
		}
	}
	remaining := candidates[start:]

	hasNext := len(remaining) > limit
	if hasNext {
		remaining = remaining[:limit]
	}
	out := make([]pain.Pain, 0, len(remaining))
	for _, t := range remaining {
		out = append(out, t.Pain)
	}
	if !hasNext || len(remaining) == 0 {
		return Page{Pains: out}, nil
	}
	last := remaining[len(remaining)-1]
	return Page{
		Pains:      out,
		HasNext:    true,
		NextCursor: encodeCursor(Trending, cursor{id: last.Pain.ID, hot: hot(last, now), hasHot: true}),
	}, nil
}

// pageOf trims the extra probe row and derives the next cursor for the SQL tabs.
func pageOf(rows []pain.Pain, limit int, tab Tab) Page {
	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}
	p := Page{Pains: rows, HasNext: hasNext}
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		p.NextCursor = encodeCursor(tab, cursor{id: last.ID, createdAt: last.CreatedAt, votes: last.VoteCount})
	}
	return p
}

func hot(t pain.Trending, now time.Time) float64 {
	// comments7d is 0 until the comments table lands in M2 (FR-FEED-2).
	return scoring.Hot(t.Votes7d, 0, t.Pain.Urgency, now.Sub(t.Pain.CreatedAt))
}

func orderFor(tab Tab) pain.FeedOrder {
	if tab == MostVoted {
		return pain.FeedTopVotes
	}
	return pain.FeedLatest
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

// normalizeCategory drops a slug that is not in the catalog, so an unknown chip
// falls back to the unfiltered feed rather than leaking an arbitrary value into
// the query.
func normalizeCategory(slug string) string {
	if _, ok := taxonomy.CategoryBySlug(slug); !ok {
		return ""
	}
	return slug
}

func normalizeSearch(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > MaxSearchLen {
		r = r[:MaxSearchLen]
	}
	return string(r)
}

// cursor is a live keyset position: typed values the service compares and hands
// to the store.
type cursor struct {
	createdAt time.Time
	id        uuid.UUID
	votes     int
	hot       float64
	hasHot    bool
}

// wireCursor is the opaque, URL-safe encoding of a cursor. Unused fields are
// omitted; a decoded cursor with a different Tab is rejected.
type wireCursor struct {
	Tab       string `json:"t"`
	CreatedAt string `json:"c,omitempty"`
	ID        string `json:"i,omitempty"`
	Votes     int    `json:"v,omitempty"`
	Hot       string `json:"h,omitempty"`
}

// startCursor is the "first page" keyset: far future with max votes so every
// published row compares less than it.
func startCursor() cursor {
	return cursor{
		id:        uuid.Nil,
		createdAt: farFuture,
		votes:     math.MaxInt32,
	}
}

func encodeCursor(tab Tab, k cursor) string {
	w := wireCursor{
		Tab:   string(tab),
		ID:    k.id.String(),
		Votes: k.votes,
	}
	if !k.createdAt.IsZero() {
		w.CreatedAt = k.createdAt.UTC().Format(time.RFC3339Nano)
	}
	if k.hasHot {
		w.Hot = strconv.FormatFloat(k.hot, 'g', 17, 64)
	}
	b, _ := json.Marshal(w)
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeCursor returns a usable keyset, falling back to the first page for an
// empty, malformed or mismatched cursor. A bad cursor is a harmless client
// detail, so it never fails the request.
func decodeCursor(raw string, tab Tab) cursor {
	if raw == "" {
		return startCursor()
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return startCursor()
	}
	var w wireCursor
	if err := json.Unmarshal(b, &w); err != nil || w.Tab != string(tab) {
		return startCursor()
	}
	k := cursor{votes: w.Votes, createdAt: farFuture}
	if id, err := uuid.Parse(w.ID); err == nil {
		k.id = id
	}
	if w.CreatedAt != "" {
		if ts, err := time.Parse(time.RFC3339Nano, w.CreatedAt); err == nil {
			k.createdAt = ts
		}
	}
	if w.Hot != "" {
		if f, err := strconv.ParseFloat(w.Hot, 64); err == nil {
			k.hot, k.hasHot = f, true
		}
	}
	return k
}

// feedCursor converts a decoded cursor to the store filter shape.
func (k cursor) feedCursor() pain.FeedCursor {
	return pain.FeedCursor{CreatedAt: k.createdAt, ID: k.id, Votes: k.votes}
}
