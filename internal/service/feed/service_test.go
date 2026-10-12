package feed

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"grumble/internal/domain/pain"
)

var feedNow = time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

type fakeStore struct {
	latest     []pain.Pain
	top        []pain.Pain
	candidates []pain.Trending
	lastLimit  int
}

func (f *fakeStore) ListFeed(_ context.Context, fl pain.FeedFilter) ([]pain.Pain, error) {
	f.lastLimit = fl.Limit - 1
	// Simulate old semantics for the existing tests.
	switch fl.Order {
	case pain.FeedTopVotes:
		return f.top, nil
	default:
		return f.latest, nil
	}
}

func (f *fakeStore) ListTrendingCandidates(_ context.Context, _, _ string, limit int) ([]pain.Trending, error) {
	f.lastLimit = limit
	return f.candidates, nil
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return feedNow }

func trend(id byte, votes7d, urgency int, age time.Duration) pain.Trending {
	u := uuid.UUID{id}
	return pain.Trending{
		Pain:    pain.Pain{ID: u, CreatedAt: feedNow.Add(-age), Urgency: urgency},
		Votes7d: votes7d,
	}
}

func TestTrendingRanksVotesThenAge(t *testing.T) {
	store := &fakeStore{candidates: []pain.Trending{
		trend(3, 10, 80, 100*time.Hour), // oldest, same votes/urgency -> last
		trend(1, 10, 80, time.Hour),     // most votes, recent -> first
		trend(2, 5, 80, time.Hour),      // fewer votes -> second
	}}
	svc := New(store, fixedClock{})

	got, err := svc.List(context.Background(), Query{Tab: Trending, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.Pains) != 3 {
		t.Fatalf("len = %d, want 3", len(got.Pains))
	}
	want := []byte{1, 2, 3}
	for i, wantID := range want {
		if got.Pains[i].ID != (uuid.UUID{wantID}) {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestTrendingPagesInMemory(t *testing.T) {
	store := &fakeStore{candidates: []pain.Trending{
		trend(1, 10, 80, time.Hour),
		trend(2, 5, 80, time.Hour),
		trend(3, 1, 80, time.Hour),
	}}
	svc := New(store, fixedClock{})

	gotFirst, err := svc.List(context.Background(), Query{Tab: Trending, Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got, err := svc.List(context.Background(), Query{Tab: Trending, Limit: 1, Cursor: gotFirst.NextCursor})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.Pains) != 1 || got.Pains[0].ID != (uuid.UUID{2}) {
		t.Fatalf("page = %v, want the second hottest only", got)
	}
}

func TestLatestUsesStoreAndDefaultLimit(t *testing.T) {
	store := &fakeStore{latest: []pain.Pain{{ID: uuid.New()}}}
	svc := New(store, fixedClock{})

	got, err := svc.List(context.Background(), Query{Tab: Latest})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got.Pains) != 1 {
		t.Fatalf("len = %d, want 1", len(got.Pains))
	}
	if store.lastLimit != DefaultLimit {
		t.Fatalf("limit = %d, want default %d", store.lastLimit, DefaultLimit)
	}
}

func TestLimitIsCapped(t *testing.T) {
	store := &fakeStore{}
	svc := New(store, fixedClock{})

	if _, err := svc.List(context.Background(), Query{Tab: MostVoted, Limit: 1000}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if store.lastLimit != MaxLimit {
		t.Fatalf("limit = %d, want cap %d", store.lastLimit, MaxLimit)
	}
}
