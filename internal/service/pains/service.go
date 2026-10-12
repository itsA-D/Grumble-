// Package pains owns submission and reading of pains: validation, duplicate
// slug selection, the PII confirmation step and the first moderation decision
// (PRD FR-SUB-1 to FR-SUB-8, FR-MOD-1 to FR-MOD-3, FR-DET-1). It declares the
// interfaces it needs; the composition root supplies the adapters.
package pains

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"grumble/internal/domain"
	"grumble/internal/domain/moderation"
	"grumble/internal/domain/pain"
	"grumble/internal/domain/pii"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/scoring"
)

// slugAttempts bounds how many numeric suffixes are tried before falling back
// to a random-ish suffix derived from the id.
const slugAttempts = 5

// DefaultBlockedTerms is the built-in spam/marker list (FR-MOD-2). Operators may
// replace it through Config; it is intentionally short and non-sensitive.
var DefaultBlockedTerms = []string{
	"free crypto",
	"click here to win",
	"buy now cheap",
}

// Config tunes the submission rules. Zero values fall back to the defaults.
type Config struct {
	// BlockedTerms replaces DefaultBlockedTerms when non-nil.
	BlockedTerms []string
	// Limits bounds the moderation rules.
	Limits moderation.Limits
}

// Store is the persistence the service needs (implemented by the postgres repo).
type Store interface {
	Create(ctx context.Context, in pain.New) (pain.Pain, error)
	SetStatus(ctx context.Context, id uuid.UUID, status pain.Status) error
	SlugExists(ctx context.Context, slug string) (bool, error)
	PublishedBySlug(ctx context.Context, slug string) (pain.Pain, error)
	// ListLatest is only needed for sitemap on this service. We keep it
	// satisfied by a keyset read that fetches from the beginning.
	ListFeed(ctx context.Context, f pain.FeedFilter) ([]pain.Pain, error)
	ListRelated(ctx context.Context, category string, excludeID uuid.UUID, limit int) ([]pain.Pain, error)
	UpdateCounters(ctx context.Context, id uuid.UUID, voteCount, urgency int) error
}

// TxRunner runs a unit of work in one transaction.
type TxRunner interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// SourceHasher derives the salted, monthly source hash for an address.
type SourceHasher interface {
	SourceHash(ip string, at time.Time) string
}

// Limiter gates repeated submissions (FR-MOD-2).
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// Verifier checks the Turnstile challenge token on submit (FR-MOD-7).
type Verifier interface {
	Verify(ctx context.Context, token string) error
}

// Clock supplies the current time.
type Clock interface {
	Now() time.Time
}

// IDGen returns a new time-ordered id.
type IDGen interface {
	New() (uuid.UUID, error)
}

// Service submits and reads pains.
type Service struct {
	store    Store
	tx       TxRunner
	hasher   SourceHasher
	limiter  Limiter
	verifier Verifier
	clock    Clock
	idgen    IDGen
	blocked  []string
	limits   moderation.Limits
}

// New builds the pains service.
func New(store Store, tx TxRunner, hasher SourceHasher, limiter Limiter, verifier Verifier, clk Clock, ids IDGen, cfg Config) *Service {
	blocked := cfg.BlockedTerms
	if blocked == nil {
		blocked = DefaultBlockedTerms
	}
	return &Service{
		store:    store,
		tx:       tx,
		hasher:   hasher,
		limiter:  limiter,
		verifier: verifier,
		clock:    clk,
		idgen:    ids,
		blocked:  blocked,
		limits:   cfg.Limits,
	}
}

// SubmitRequest is one submission as it arrives from the form.
type SubmitRequest struct {
	Body          string
	CategorySlug  string
	CountryCode   string
	Frequency     string
	Disruption    string
	WouldPay      bool
	MonthlyAmount int
	Tried         string
	ClientIP      string
	AnonID        string
	// TurnstileToken is the challenge token when Turnstile is enabled (FR-MOD-7).
	TurnstileToken string
	// ConfirmedRedaction is true when the submitter saw the personal-data
	// warning and chose to post the redacted version anyway (FR-MOD-3).
	ConfirmedRedaction bool
}

// SubmitResult reports the stored pain and the decision that admitted it.
type SubmitResult struct {
	Pain   pain.Pain
	Status pain.Status
}

// PersonalDataError means the body looks like it contains personal data and the
// submitter has not yet confirmed redaction. The handler re-renders the form
// with Preview so the user can confirm (FR-MOD-3).
type PersonalDataError struct {
	Kinds   []pii.Kind
	Preview string
}

func (e *PersonalDataError) Error() string { return "submission may contain personal data" }

// Is matches the shared validation sentinel so the transport treats it as a
// recoverable, user-fixable error rather than a server fault.
func (e *PersonalDataError) Is(target error) bool { return target == domain.ErrValidation }

// Submit validates, moderates and stores a new pain (FR-SUB-1 to FR-SUB-8).
func (s *Service) Submit(ctx context.Context, req SubmitRequest) (SubmitResult, error) {
	if err := s.verifier.Verify(ctx, req.TurnstileToken); err != nil {
		return SubmitResult{}, fmt.Errorf("turnstile: %w", domain.ErrForbidden)
	}

	cat, ok := taxonomy.CategoryBySlug(strings.TrimSpace(req.CategorySlug))
	if !ok {
		return SubmitResult{}, &domain.ValidationError{Field: "category", Message: "choose a category"}
	}
	country, ok := taxonomy.CountryByCode(strings.ToUpper(strings.TrimSpace(req.CountryCode)))
	if !ok {
		return SubmitResult{}, &domain.ValidationError{Field: "country", Message: "choose a country"}
	}

	draft := pain.Pain{
		Body:     req.Body,
		Category: cat,
		Country:  country,
		Details: pain.Details{
			Frequency:     pain.Frequency(req.Frequency),
			Disruption:    pain.Disruption(req.Disruption),
			WouldPay:      req.WouldPay,
			MonthlyAmount: req.MonthlyAmount,
			Tried:         req.Tried,
		},
	}
	if err := pain.Validate(draft); err != nil {
		return SubmitResult{}, normalizeValidation(err)
	}

	body := req.Body
	if findings := pii.Detect(body); len(findings) > 0 && !req.ConfirmedRedaction {
		return SubmitResult{}, &PersonalDataError{Kinds: kindsOf(findings), Preview: pii.Redact(body)}
	} else if len(findings) > 0 {
		body = pii.Redact(body)
	}

	now := s.clock.Now()
	sourceHash := s.hasher.SourceHash(req.ClientIP, now)

	allowed, err := s.limiter.Allow(ctx, "submit:"+sourceHash)
	if err != nil {
		return SubmitResult{}, fmt.Errorf("rate limit: %w", err)
	}
	if !allowed {
		return SubmitResult{}, domain.ErrRateLimited
	}

	id, err := s.idgen.New()
	if err != nil {
		return SubmitResult{}, err
	}
	slug, err := s.uniqueSlug(ctx, pain.Slugify(body), id)
	if err != nil {
		return SubmitResult{}, err
	}

	decision := moderation.Evaluate(body, s.blocked, s.limits)
	status := statusFor(decision.Decision)
	urgency := scoring.Urgency(signalsFor(draft.Details, 0))

	stored := pain.Pain{
		ID:        id,
		Slug:      slug,
		Body:      body,
		Category:  cat,
		Country:   country,
		Details:   draft.Details,
		Status:    status,
		Urgency:   urgency,
		CreatedAt: now,
	}
	if status == pain.StatusPublished {
		published := now
		stored.PublishedAt = &published
	}

	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		created, err := s.store.Create(ctx, pain.New{
			ID:             id,
			Slug:           slug,
			Body:           body,
			Category:       cat,
			Country:        country,
			Details:        draft.Details,
			SourceHash:     sourceHash,
			AnonID:         req.AnonID,
			ScoringVersion: scoring.Version,
		})
		if err != nil {
			return err
		}
		if err := s.store.SetStatus(ctx, id, status); err != nil {
			return err
		}
		if err := s.store.UpdateCounters(ctx, id, 0, urgency); err != nil {
			return err
		}
		stored.CreatedAt = created.CreatedAt
		return nil
	})
	if err != nil {
		return SubmitResult{}, err
	}
	return SubmitResult{Pain: stored, Status: status}, nil
}

// Detail is one pain plus the related pains shown beside it (FR-DET-1).
type Detail struct {
	Pain    pain.Pain
	Related []pain.Pain
}

// relatedLimit is how many related pains the detail page shows.
const relatedLimit = 6

// Get returns a published pain by slug and related pains in its category.
func (s *Service) Get(ctx context.Context, slug string) (Detail, error) {
	p, err := s.store.PublishedBySlug(ctx, slug)
	if err != nil {
		return Detail{}, err
	}
	related, err := s.store.ListRelated(ctx, p.Category.Slug, p.ID, relatedLimit)
	if err != nil {
		return Detail{}, err
	}
	return Detail{Pain: p, Related: related}, nil
}

// RecentForSitemap returns recently published pains for the sitemap. Only
// published rows are read (the store reads the published view), so held and
// rejected submissions never appear (FR-MOD-8, ADR 0001).
func (s *Service) RecentForSitemap(ctx context.Context, limit int) ([]pain.Pain, error) {
	return s.store.ListFeed(ctx, pain.FeedFilter{
		Order:    pain.FeedLatest,
		Category: "",
		Search:   "",
		After:    pain.FeedCursor{CreatedAt: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), ID: uuid.Nil, Votes: 0},
		Limit:    limit + 1,
	})
}

// uniqueSlug returns base, or base with a suffix, that no pain uses yet
// (FR-SUB-8).
func (s *Service) uniqueSlug(ctx context.Context, base string, id uuid.UUID) (string, error) {
	if base == "" {
		base = "pain"
	}
	candidate := base
	for i := 0; i < slugAttempts; i++ {
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		exists, err := s.store.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return base + "-" + id.String()[:8], nil
}

// normalizeValidation converts a pain-package field error into the shared
// domain.ValidationError so the transport has one validation type to map
// (CLAUDE.md error conventions).
func normalizeValidation(err error) error {
	var ve pain.ValidationError
	if errors.As(err, &ve) {
		return &domain.ValidationError{Field: ve.Field, Message: ve.Message}
	}
	return err
}

// statusFor maps the automated decision to the initial status (FR-MOD-1).
func statusFor(d moderation.Decision) pain.Status {
	switch d {
	case moderation.Reject:
		return pain.StatusRejected
	case moderation.Hold:
		return pain.StatusHeld
	default:
		return pain.StatusPublished
	}
}

// signalsFor projects stored details plus a vote count into the pure scoring
// input (PRD section 9.1). The conversion lives in the service so domain stays
// free of a scoring dependency.
func signalsFor(d pain.Details, votes int) scoring.Signals {
	return scoring.Signals{
		Frequency:     string(d.Frequency),
		Disruption:    string(d.Disruption),
		WouldPay:      d.WouldPay,
		MonthlyAmount: d.MonthlyAmount,
		Votes:         votes,
	}
}

func kindsOf(findings []pii.Finding) []pii.Kind {
	seen := make(map[pii.Kind]bool, len(findings))
	out := make([]pii.Kind, 0, len(findings))
	for _, f := range findings {
		if !seen[f.Kind] {
			seen[f.Kind] = true
			out = append(out, f.Kind)
		}
	}
	return out
}
