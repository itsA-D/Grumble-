package voting

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
	"grumble/internal/scoring"
)

type fakeStore struct {
	ctx       pain.VoteContext
	lockErr   error
	votes     int
	urgency   int
	updated   bool
	updateErr error
}

func (f *fakeStore) LockForVote(context.Context, uuid.UUID) (pain.VoteContext, error) {
	return f.ctx, f.lockErr
}

func (f *fakeStore) UpdateCounters(_ context.Context, _ uuid.UUID, votes, urgency int) error {
	f.votes, f.urgency, f.updated = votes, urgency, true
	return f.updateErr
}

type fakeVotes struct {
	set map[string]bool
}

func newFakeVotes() *fakeVotes { return &fakeVotes{set: map[string]bool{}} }

func (f *fakeVotes) HasVoted(_ context.Context, id uuid.UUID, key string) (bool, error) {
	return f.set[id.String()+"|"+key], nil
}

func (f *fakeVotes) Add(_ context.Context, id uuid.UUID, key string) (bool, error) {
	k := id.String() + "|" + key
	if f.set[k] {
		return false, nil
	}
	f.set[k] = true
	return true, nil
}

func (f *fakeVotes) Remove(_ context.Context, id uuid.UUID, key string) (bool, error) {
	k := id.String() + "|" + key
	if !f.set[k] {
		return false, nil
	}
	delete(f.set, k)
	return true, nil
}

func (f *fakeVotes) Count(_ context.Context, id uuid.UUID) (int, error) {
	n := 0
	for k := range f.set {
		if len(k) > len(id.String()) && k[:len(id.String())] == id.String() {
			n++
		}
	}
	return n, nil
}

type fakeTx struct{}

func (fakeTx) InTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func publishedCtx() pain.VoteContext {
	return pain.VoteContext{
		ID:            uuid.New(),
		Status:        pain.StatusPublished,
		Frequency:     pain.FrequencyDaily,
		Disruption:    pain.DisruptionAnnoying,
		WouldPay:      true,
		MonthlyAmount: 10,
	}
}

func TestSetAddsVoteAndRecomputesUrgency(t *testing.T) {
	store := &fakeStore{ctx: publishedCtx()}
	votes := newFakeVotes()
	svc := New(store, votes, fakeTx{})

	res, err := svc.Set(context.Background(), Request{PainID: store.ctx.ID, VoterKey: "a:1", Voted: true})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !res.Voted || res.VoteCount != 1 {
		t.Fatalf("result = %+v, want voted with count 1", res)
	}
	wantUrgency := scoring.Urgency(scoring.Signals{
		Frequency: "daily", Disruption: "annoying", WouldPay: true, MonthlyAmount: 10, Votes: 1,
	})
	if res.Urgency != wantUrgency || store.urgency != wantUrgency {
		t.Fatalf("urgency = %d / stored %d, want %d", res.Urgency, store.urgency, wantUrgency)
	}
	if !store.updated {
		t.Fatal("counters not updated")
	}
}

func TestSetIsIdempotent(t *testing.T) {
	store := &fakeStore{ctx: publishedCtx()}
	votes := newFakeVotes()
	svc := New(store, votes, fakeTx{})
	req := Request{PainID: store.ctx.ID, VoterKey: "a:1", Voted: true}

	if _, err := svc.Set(context.Background(), req); err != nil {
		t.Fatalf("first Set: %v", err)
	}
	res, err := svc.Set(context.Background(), req)
	if err != nil {
		t.Fatalf("second Set: %v", err)
	}
	if res.VoteCount != 1 {
		t.Fatalf("count = %d, want 1 (repeat must not double-count)", res.VoteCount)
	}
}

func TestSetRemovesVote(t *testing.T) {
	store := &fakeStore{ctx: publishedCtx()}
	votes := newFakeVotes()
	svc := New(store, votes, fakeTx{})
	painID := store.ctx.ID

	if _, err := svc.Set(context.Background(), Request{PainID: painID, VoterKey: "a:1", Voted: true}); err != nil {
		t.Fatalf("add: %v", err)
	}
	res, err := svc.Set(context.Background(), Request{PainID: painID, VoterKey: "a:1", Voted: false})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if res.Voted || res.VoteCount != 0 {
		t.Fatalf("result = %+v, want unvoted with count 0", res)
	}
}

func TestToggleFlips(t *testing.T) {
	store := &fakeStore{ctx: publishedCtx()}
	svc := New(store, newFakeVotes(), fakeTx{})
	painID := store.ctx.ID

	first, err := svc.Toggle(context.Background(), painID, "a:1")
	if err != nil {
		t.Fatalf("toggle 1: %v", err)
	}
	if !first.Voted || first.VoteCount != 1 {
		t.Fatalf("first toggle = %+v, want voted", first)
	}
	second, err := svc.Toggle(context.Background(), painID, "a:1")
	if err != nil {
		t.Fatalf("toggle 2: %v", err)
	}
	if second.Voted || second.VoteCount != 0 {
		t.Fatalf("second toggle = %+v, want unvoted", second)
	}
}

func TestSetRejectsUnpublished(t *testing.T) {
	ctx := publishedCtx()
	ctx.Status = pain.StatusHeld
	store := &fakeStore{ctx: ctx}
	votes := newFakeVotes()
	svc := New(store, votes, fakeTx{})

	_, err := svc.Set(context.Background(), Request{PainID: ctx.ID, VoterKey: "a:1", Voted: true})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if len(votes.set) != 0 {
		t.Fatal("vote recorded on unpublished pain")
	}
	if store.updated {
		t.Fatal("counters updated for unpublished pain")
	}
}
