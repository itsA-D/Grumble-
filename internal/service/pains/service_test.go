package pains

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
	"grumble/internal/scoring"
)

var testInstant = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

type fakeStore struct {
	created   pain.New
	status    pain.Status
	votes     int
	urgency   int
	haveCount bool
	slugTaken map[string]bool
	err       error
}

func (f *fakeStore) Create(_ context.Context, in pain.New) (pain.Pain, error) {
	f.created = in
	return pain.Pain{ID: in.ID, Slug: in.Slug, CreatedAt: testInstant}, f.err
}

func (f *fakeStore) SetStatus(_ context.Context, _ uuid.UUID, s pain.Status) error {
	f.status = s
	return f.err
}

func (f *fakeStore) SlugExists(_ context.Context, slug string) (bool, error) {
	return f.slugTaken[slug], f.err
}

func (f *fakeStore) PublishedBySlug(_ context.Context, _ string) (pain.Pain, error) {
	return pain.Pain{}, domain.ErrNotFound
}

func (f *fakeStore) ListLatest(_ context.Context, _, _ int) ([]pain.Pain, error) {
	return nil, f.err
}

func (f *fakeStore) ListRelated(_ context.Context, _ string, _ uuid.UUID, _ int) ([]pain.Pain, error) {
	return nil, f.err
}

func (f *fakeStore) UpdateCounters(_ context.Context, _ uuid.UUID, votes, urgency int) error {
	f.votes, f.urgency, f.haveCount = votes, urgency, true
	return f.err
}

type fakeTx struct{ err error }

func (f *fakeTx) InTx(ctx context.Context, fn func(context.Context) error) error {
	if f.err != nil {
		return f.err
	}
	return fn(ctx)
}

type fakeHasher struct{ hash string }

func (f fakeHasher) SourceHash(string, time.Time) string { return f.hash }

type fakeLimiter struct {
	allow bool
	err   error
}

func (f fakeLimiter) Allow(context.Context, string) (bool, error) { return f.allow, f.err }

type fixedClock struct{}

func (fixedClock) Now() time.Time { return testInstant }

type fixedIDs struct{ id uuid.UUID }

func (f fixedIDs) New() (uuid.UUID, error) { return f.id, nil }

// okVerifier accepts every token (Turnstile disabled in tests).
type okVerifier struct{ err error }

func (v okVerifier) Verify(context.Context, string) error { return v.err }

func newService(store Store, hasher SourceHasher, limiter Limiter, cfg Config) *Service {
	return New(store, &fakeTx{}, hasher, limiter, okVerifier{}, fixedClock{}, fixedIDs{id: uuid.New()}, cfg)
}

func baseRequest() SubmitRequest {
	return SubmitRequest{
		Body:         "The buses never arrive on time and nobody explains why.",
		CategorySlug: "transportation",
		CountryCode:  "GB",
		ClientIP:     "203.0.113.7",
		AnonID:       "anon-1",
	}
}

func TestSubmitPublishesCleanPain(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})

	res, err := svc.Submit(context.Background(), baseRequest())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if res.Status != pain.StatusPublished {
		t.Fatalf("status = %s, want published", res.Status)
	}
	if store.status != pain.StatusPublished {
		t.Fatalf("stored status = %s, want published", store.status)
	}
	if res.Pain.Slug == "" || store.created.Slug != res.Pain.Slug {
		t.Fatalf("slug not persisted: %q vs %q", res.Pain.Slug, store.created.Slug)
	}
	if store.created.SourceHash != "h" {
		t.Fatalf("source hash = %q, want h", store.created.SourceHash)
	}
	if !store.haveCount || store.votes != 0 {
		t.Fatalf("counters = (%d,%d), want votes 0", store.votes, store.urgency)
	}
	if res.Pain.PublishedAt == nil {
		t.Fatal("published pain has no PublishedAt")
	}
}

func TestSubmitComputesUrgency(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})
	req := baseRequest()
	req.Frequency = "daily"
	req.Disruption = "annoying"
	req.WouldPay = true
	req.MonthlyAmount = 10

	res, err := svc.Submit(context.Background(), req)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	// PRD 9.2 for daily + annoying + $10 + no votes = 64.
	if res.Pain.Urgency != 64 {
		t.Fatalf("urgency = %d, want 64", res.Pain.Urgency)
	}
	if store.urgency != 64 {
		t.Fatalf("stored urgency = %d, want 64", store.urgency)
	}
}

func TestSubmitRejectsBlockedTerm(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})
	req := baseRequest()
	req.Body = "This whole offer smells like free crypto nonsense to me."

	res, err := svc.Submit(context.Background(), req)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if res.Status != pain.StatusRejected {
		t.Fatalf("status = %s, want rejected", res.Status)
	}
}

func TestSubmitHoldsTooManyLinks(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})
	req := baseRequest()
	req.Body = "See https://a.example and https://b.example and https://c.example for proof."

	res, err := svc.Submit(context.Background(), req)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if res.Status != pain.StatusHeld {
		t.Fatalf("status = %s, want held", res.Status)
	}
}

func TestSubmitAsksToConfirmPersonalData(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})
	req := baseRequest()
	req.Body = "They keep emailing me at jane.doe@example.com about this problem."

	_, err := svc.Submit(context.Background(), req)
	var pd *PersonalDataError
	if !errors.As(err, &pd) {
		t.Fatalf("err = %v, want *PersonalDataError", err)
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatal("PersonalDataError should match ErrValidation")
	}
	if strings.Contains(pd.Preview, "jane.doe@example.com") || !strings.Contains(pd.Preview, "[email]") {
		t.Fatalf("preview not redacted: %q", pd.Preview)
	}
	if store.created.Body != "" {
		t.Fatal("pain stored before confirmation")
	}
}

func TestSubmitRedactsAfterConfirmation(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})
	req := baseRequest()
	req.Body = "They keep emailing me at jane.doe@example.com about this problem."
	req.ConfirmedRedaction = true

	res, err := svc.Submit(context.Background(), req)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if strings.Contains(res.Pain.Body, "jane.doe@example.com") {
		t.Fatalf("stored body leaked email: %q", res.Pain.Body)
	}
	if !strings.Contains(res.Pain.Body, "[email]") {
		t.Fatalf("stored body missing placeholder: %q", res.Pain.Body)
	}
}

func TestSubmitRateLimited(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: false}, Config{})

	_, err := svc.Submit(context.Background(), baseRequest())
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
	if store.created.Body != "" {
		t.Fatal("pain stored despite rate limit")
	}
}

func TestSubmitTurnstileFailureForbidden(t *testing.T) {
	store := &fakeStore{}
	svc := New(store, &fakeTx{}, fakeHasher{"h"}, fakeLimiter{allow: true}, okVerifier{err: errors.New("bad token")}, fixedClock{}, fixedIDs{id: uuid.New()}, Config{})

	_, err := svc.Submit(context.Background(), baseRequest())
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
	if store.created.Body != "" {
		t.Fatal("pain stored despite failed challenge")
	}
}

func TestSubmitHoldsSelfHarm(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})
	req := baseRequest()
	req.Body = "I keep thinking I want to die and nothing seems to help anymore."

	res, err := svc.Submit(context.Background(), req)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if res.Status != pain.StatusHeld {
		t.Fatalf("status = %s, want held (self-harm must never publish)", res.Status)
	}
}

func TestSubmitValidationErrors(t *testing.T) {
	svc := newService(&fakeStore{}, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})

	req := baseRequest()
	req.CategorySlug = "nonsense"
	if _, err := svc.Submit(context.Background(), req); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad category err = %v, want validation", err)
	}

	req = baseRequest()
	req.CountryCode = "ZZ"
	if _, err := svc.Submit(context.Background(), req); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad country err = %v, want validation", err)
	}

	req = baseRequest()
	req.Body = "too short"
	if _, err := svc.Submit(context.Background(), req); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("short body err = %v, want validation", err)
	}
}

func TestUniqueSlugSuffixesOnCollision(t *testing.T) {
	store := &fakeStore{slugTaken: map[string]bool{"transport": true}}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})
	req := baseRequest()
	req.Body = "Transport transport transport everywhere you look around here."

	got, err := svc.uniqueSlug(context.Background(), "transport", uuid.New())
	if err != nil {
		t.Fatalf("uniqueSlug: %v", err)
	}
	if got == "transport" {
		t.Fatalf("slug %q was already taken", got)
	}
}

func TestVersionMatchesScoring(t *testing.T) {
	store := &fakeStore{}
	svc := newService(store, fakeHasher{"h"}, fakeLimiter{allow: true}, Config{})
	if _, err := svc.Submit(context.Background(), baseRequest()); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if store.created.ScoringVersion != scoring.Version {
		t.Fatalf("scoring version = %d, want %d", store.created.ScoringVersion, scoring.Version)
	}
}
