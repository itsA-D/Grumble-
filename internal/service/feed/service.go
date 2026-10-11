// Package feed assembles the three feed tabs: Latest, Most voted and Trending
// (PRD FR-FEED-1, FR-FEED-2). Latest and Most voted are ordered in SQL; Trending
// is finished here with the pure hot rank so the formula lives only in scoring.
package feed

import (
	"context"
	"sort"
	"time"

	"grumble/internal/domain/pain"
	"grumble/internal/scoring"
)

// Page-size bounds.
const (
	DefaultLimit = 20
	MaxLimit     = 100
	// candidateLimit bounds the recent set the hot rank sorts in Go. Phase 2
	// replaces this with a maintained read model.
	candidateLimit = 200
)

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
	ListLatest(ctx context.Context, limit, offset int) ([]pain.Pain, error)
	ListTopVotes(ctx context.Context, limit, offset int) ([]pain.Pain, error)
	ListTrendingCandidates(ctx context.Context, limit int) ([]pain.Trending, error)
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

// Query is one feed page request.
type Query struct {
	Tab    Tab
	Limit  int
	Offset int
}

// List returns one page of the requested tab.
func (s *Service) List(ctx context.Context, q Query) ([]pain.Pain, error) {
	limit, offset := normalize(q)
	switch q.Tab {
	case MostVoted:
		return s.store.ListTopVotes(ctx, limit, offset)
	case Trending:
		return s.listTrending(ctx, limit, offset)
	default:
		return s.store.ListLatest(ctx, limit, offset)
	}
}

// listTrending ranks recent candidates by the pure hot score, then pages the
// sorted slice in memory.
func (s *Service) listTrending(ctx context.Context, limit, offset int) ([]pain.Pain, error) {
	candidates, err := s.store.ListTrendingCandidates(ctx, candidateLimit)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	sort.SliceStable(candidates, func(i, j int) bool {
		return hot(candidates[i], now) > hot(candidates[j], now)
	})
	return page(candidates, limit, offset), nil
}

func hot(t pain.Trending, now time.Time) float64 {
	// comments7d is 0 until the comments table lands in M2 (FR-FEED-2).
	return scoring.Hot(t.Votes7d, 0, t.Pain.Urgency, now.Sub(t.Pain.CreatedAt))
}

func page(items []pain.Trending, limit, offset int) []pain.Pain {
	if offset >= len(items) {
		return nil
	}
	items = items[offset:]
	if limit > len(items) {
		limit = len(items)
	}
	out := make([]pain.Pain, 0, limit)
	for _, t := range items[:limit] {
		out = append(out, t.Pain)
	}
	return out
}

func normalize(q Query) (limit, offset int) {
	limit = q.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	offset = q.Offset
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
