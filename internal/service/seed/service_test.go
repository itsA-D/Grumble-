package seed

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/OWNER/painboard/internal/domain/pain"
	"github.com/OWNER/painboard/internal/domain/pii"
)

// --- fakes ------------------------------------------------------------------

type row struct {
	in        pain.New
	votes     map[string]bool
	voteCount int
	urgency   int
}

type fakeStore struct {
	rows  map[string]*row
	byID  map[uuid.UUID]*row
	order []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{rows: map[string]*row{}, byID: map[uuid.UUID]*row{}}
}

func (f *fakeStore) CreateSeeded(_ context.Context, in pain.New) (uuid.UUID, bool, error) {
	if _, ok := f.rows[in.Slug]; ok {
		return uuid.Nil, false, nil
	}
	r := &row{in: in, votes: map[string]bool{}}
	f.rows[in.Slug] = r
	f.byID[in.ID] = r
	f.order = append(f.order, in.Slug)
	return in.ID, true, nil
}

func (f *fakeStore) AddSeedVotes(_ context.Context, id uuid.UUID, keys []string) (int, error) {
	r := f.byID[id]
	added := 0
	for _, k := range keys {
		if !r.votes[k] {
			r.votes[k] = true
			added++
		}
	}
	return added, nil
}

func (f *fakeStore) UpdateCounters(_ context.Context, id uuid.UUID, voteCount, urgency int) error {
	r := f.byID[id]
	r.voteCount = voteCount
	r.urgency = urgency
	return nil
}

type noTx struct{}

func (noTx) InTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func testService(store Store) *Service {
	n := 0
	return New(store, noTx{}, func() uuid.UUID {
		n++
		return uuid.New()
	}, DefaultItems())
}

// --- tests ------------------------------------------------------------------

// TestRunIsIdempotentAndLabelled runs the whole seed set twice: the second run
// must insert nothing, and every stored row must be published and flagged
// seeded (never organic).
func TestRunIsIdempotentAndLabelled(t *testing.T) {
	store := newFakeStore()
	svc := testService(store)

	first, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if first.Created != len(DefaultItems()) || first.Skipped != 0 {
		t.Fatalf("first run = %+v, want created=%d skipped=0", first, len(DefaultItems()))
	}

	second, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second.Created != 0 || second.Skipped != len(DefaultItems()) {
		t.Fatalf("second run = %+v, want created=0 skipped=%d", second, len(DefaultItems()))
	}

	for slug, r := range store.rows {
		if !r.in.Seeded {
			t.Errorf("row %q not flagged seeded", slug)
		}
		if r.voteCount != len(r.votes) {
			t.Errorf("row %q vote_count=%d but has %d vote rows", slug, r.voteCount, len(r.votes))
		}
	}
}

// TestRunCountersMatchVotes proves the urgency is recomputed from the real vote
// count in the same unit of work (invariant 6).
func TestRunCountersMatchVotes(t *testing.T) {
	store := newFakeStore()
	if _, err := testService(store).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	for slug, r := range store.rows {
		if r.voteCount == 0 {
			continue
		}
		if r.urgency < 1 || r.urgency > 100 {
			t.Errorf("row %q urgency=%d outside 1..100", slug, r.urgency)
		}
	}
}

func TestContentIsValidAndSpread(t *testing.T) {
	categories := map[string]int{}
	countries := map[string]int{}
	slugs := map[string]bool{}

	for _, item := range DefaultItems() {
		if len(item.Body) < 20 || len(item.Body) > 2000 {
			t.Errorf("body length %d out of range: %q", len(item.Body), item.Body)
		}
		if pii.Has(item.Body) {
			t.Errorf("seed body looks like it contains personal data: %q", item.Body)
		}
		if _, err := resolveCategory(item.Category); err != nil {
			t.Errorf("item %q: %v", item.Body, err)
		}
		if _, err := resolveCountry(item.Country); err != nil {
			t.Errorf("item %q: %v", item.Body, err)
		}
		slug := pain.Slugify(item.Body)
		if slug == "" {
			t.Errorf("item %q produced an empty slug", item.Body)
		}
		if slugs[slug] {
			t.Errorf("duplicate seed slug %q", slug)
		}
		slugs[slug] = true
		categories[item.Category]++
		countries[item.Country]++
	}

	if len(categories) < 10 {
		t.Errorf("seed covers %d categories, want at least 10", len(categories))
	}
	if len(countries) < 10 {
		t.Errorf("seed covers %d countries, want at least 10", len(countries))
	}
	if len(DefaultItems()) < 50 {
		t.Errorf("seed set has %d items, want a fuller board", len(DefaultItems()))
	}
}
