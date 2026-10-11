package hubs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
)

type fakeStore struct {
	catCount   int
	countryCnt int
	pains      []pain.Pain
	err        error
}

func (f *fakeStore) CountByCategory(context.Context, string) (int, error) { return f.catCount, f.err }
func (f *fakeStore) CountByCountry(context.Context, string) (int, error)  { return f.countryCnt, f.err }

func (f *fakeStore) ListByCategory(context.Context, string, int, int) ([]pain.Pain, error) {
	return f.pains, f.err
}

func (f *fakeStore) ListByCountry(context.Context, string, int, int) ([]pain.Pain, error) {
	return f.pains, f.err
}

func TestCategoryHubIndexableAtFloor(t *testing.T) {
	svc := New(&fakeStore{catCount: MinPublishedForIndex})

	hub, err := svc.Get(context.Background(), Query{Kind: Category, Slug: "tech"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hub.Indexable {
		t.Fatalf("hub with %d pains should be indexable", MinPublishedForIndex)
	}
	if hub.Title == "" || hub.Intro == "" || len(hub.FAQ) == 0 {
		t.Fatal("hub copy is incomplete")
	}
}

func TestCategoryHubNoindexBelowFloor(t *testing.T) {
	svc := New(&fakeStore{catCount: MinPublishedForIndex - 1})

	hub, err := svc.Get(context.Background(), Query{Kind: Category, Slug: "tech"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hub.Indexable {
		t.Fatalf("hub with %d pains must not be indexable", hub.Count)
	}
}

func TestDescriptionNeverSaysZero(t *testing.T) {
	if strings.Contains(description(0, "frustrations about Tech"), "0") {
		t.Fatalf("zero-count description mentions a number: %q", description(0, "frustrations about Tech"))
	}
	if !strings.Contains(description(7, "frustrations about Tech"), "7") {
		t.Fatalf("non-zero description should state the count: %q", description(7, "frustrations about Tech"))
	}
}

func TestUnknownHubNotFound(t *testing.T) {
	svc := New(&fakeStore{})
	if _, err := svc.Get(context.Background(), Query{Kind: Category, Slug: "nonsense"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown category err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(context.Background(), Query{Kind: Country, Slug: "ZZ"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown country err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(context.Background(), Query{Kind: Kind("weird"), Slug: "tech"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown kind err = %v, want ErrNotFound", err)
	}
}

func TestCountryHubCopy(t *testing.T) {
	svc := New(&fakeStore{countryCnt: 3})
	hub, err := svc.Get(context.Background(), Query{Kind: Country, Slug: "GB"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hub.Title != "Frustrations in United Kingdom" {
		t.Fatalf("title = %q", hub.Title)
	}
	if hub.Indexable {
		t.Fatal("country hub with 3 pains must not be indexable")
	}
}
