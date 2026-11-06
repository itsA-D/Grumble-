package bookmarks

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
)

type fakeStore struct {
	saved     map[uuid.UUID]bool
	published map[uuid.UUID]bool
	list      []pain.Pain
}

func newStore() *fakeStore {
	return &fakeStore{saved: map[uuid.UUID]bool{}, published: map[uuid.UUID]bool{}}
}

func (f *fakeStore) Add(_ context.Context, _ string, id uuid.UUID) (bool, error) {
	if !f.published[id] || f.saved[id] {
		return false, nil
	}
	f.saved[id] = true
	return true, nil
}

func (f *fakeStore) Remove(_ context.Context, _ string, id uuid.UUID) error {
	delete(f.saved, id)
	return nil
}

func (f *fakeStore) Has(_ context.Context, _ string, id uuid.UUID) (bool, error) {
	return f.saved[id], nil
}

func (f *fakeStore) Count(context.Context, string) (int, error) { return len(f.saved), nil }

func (f *fakeStore) IDs(context.Context, string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(f.saved))
	for id := range f.saved {
		out = append(out, id)
	}
	return out, nil
}

func (f *fakeStore) List(context.Context, string, int) ([]pain.Pain, error) { return f.list, nil }

type noTx struct{}

func (noTx) InTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func TestSaveIsIdempotent(t *testing.T) {
	store := newStore()
	id := uuid.New()
	store.published[id] = true
	svc := New(store, noTx{}, Config{SavedLimit: 5})

	first, err := svc.Save(context.Background(), "a:1", id)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	second, err := svc.Save(context.Background(), "a:1", id)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if !first.Saved || !second.Saved || second.Count != 1 {
		t.Fatalf("result = %+v / %+v, want one saved row", first, second)
	}
}

func TestSaveRejectsAtCap(t *testing.T) {
	store := newStore()
	svc := New(store, noTx{}, Config{SavedLimit: 1})
	a, b := uuid.New(), uuid.New()
	store.published[a], store.published[b] = true, true

	if _, err := svc.Save(context.Background(), "a:1", a); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if _, err := svc.Save(context.Background(), "a:1", b); !errors.Is(err, ErrLimit) {
		t.Fatalf("second save error = %v, want ErrLimit", err)
	}
}

func TestSaveUnpublishedIsNotFound(t *testing.T) {
	svc := New(newStore(), noTx{}, Config{SavedLimit: 5})
	if _, err := svc.Save(context.Background(), "a:1", uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want not found", err)
	}
}

func TestRemoveReturnsNewCount(t *testing.T) {
	store := newStore()
	id := uuid.New()
	store.published[id] = true
	store.saved[id] = true
	svc := New(store, noTx{}, Config{SavedLimit: 5})

	got, err := svc.Remove(context.Background(), "a:1", id)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if got.Saved || got.Count != 0 {
		t.Fatalf("result = %+v, want unsaved with zero", got)
	}
}
