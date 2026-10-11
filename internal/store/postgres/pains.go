package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OWNER/painboard/internal/domain"
	"github.com/OWNER/painboard/internal/domain/pain"
	"github.com/OWNER/painboard/internal/store/postgres/queries"
)

// Pains is the repository behind every pain read and write. Public reads go
// through the published_pains view (ADR 0001); nothing here returns an
// unpublished row through the read methods. It speaks domain types so services
// never see database shapes (ARCHITECTURE.md boundary 9).
type Pains struct {
	pool *pgxpool.Pool
}

// NewPains builds the pain repository on a pool.
func NewPains(pool *pgxpool.Pool) *Pains {
	return &Pains{pool: pool}
}

// Create inserts a pending pain (FR-SUB-1); the caller sets the final status.
func (r *Pains) Create(ctx context.Context, in pain.New) (pain.Pain, error) {
	row, err := queries.New(handle(ctx, r.pool)).CreatePain(ctx, queries.CreatePainParams{
		ID:             in.ID,
		Slug:           in.Slug,
		Body:           in.Body,
		Category:       in.Category.Slug,
		Country:        in.Country.Code,
		Frequency:      optionalString(string(in.Details.Frequency)),
		Disruption:     optionalString(string(in.Details.Disruption)),
		WouldPay:       in.Details.WouldPay,
		MonthlyAmount:  optionalInt32(in.Details.MonthlyAmount),
		Tried:          optionalString(in.Details.Tried),
		SourceHash:     in.SourceHash,
		AnonID:         optionalString(in.AnonID),
		Seeded:         in.Seeded,
		ScoringVersion: int32(in.ScoringVersion),
	})
	if err != nil {
		return pain.Pain{}, err
	}
	return painFromModel(row), nil
}

// SetStatus moves a pain to its moderation outcome (FR-MOD-1).
func (r *Pains) SetStatus(ctx context.Context, id uuid.UUID, status pain.Status) error {
	return queries.New(handle(ctx, r.pool)).SetPainStatus(ctx, queries.SetPainStatusParams{
		ID:     id,
		Status: string(status),
	})
}

// UpdateCounters writes vote_count and the urgency score computed by scoring,
// inside the change's transaction (invariant 6).
func (r *Pains) UpdateCounters(ctx context.Context, id uuid.UUID, voteCount, urgency int) error {
	return queries.New(handle(ctx, r.pool)).UpdatePainCounters(ctx, queries.UpdatePainCountersParams{
		ID:        id,
		VoteCount: int32(voteCount),
		Urgency:   int32(urgency),
	})
}

// SlugExists reports whether a slug is already taken (FR-SUB-8).
func (r *Pains) SlugExists(ctx context.Context, slug string) (bool, error) {
	return queries.New(handle(ctx, r.pool)).PainSlugExists(ctx, slug)
}

// LockForVote locks and returns the pain row inside the current transaction, so
// concurrent votes on one pain serialise. Returns domain.ErrNotFound when
// absent; the caller enforces the published check.
func (r *Pains) LockForVote(ctx context.Context, id uuid.UUID) (pain.VoteContext, error) {
	row, err := queries.New(handle(ctx, r.pool)).GetPainForVote(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return pain.VoteContext{}, domain.ErrNotFound
	}
	if err != nil {
		return pain.VoteContext{}, err
	}
	return pain.VoteContext{
		ID:            row.ID,
		Status:        pain.Status(row.Status),
		VoteCount:     int(row.VoteCount),
		Frequency:     pain.Frequency(deref(row.Frequency)),
		Disruption:    pain.Disruption(deref(row.Disruption)),
		WouldPay:      row.WouldPay,
		MonthlyAmount: int(derefInt32(row.MonthlyAmount)),
	}, nil
}

// PublishedBySlug returns one published pain or domain.ErrNotFound.
func (r *Pains) PublishedBySlug(ctx context.Context, slug string) (pain.Pain, error) {
	row, err := queries.New(handle(ctx, r.pool)).GetPublishedBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return pain.Pain{}, domain.ErrNotFound
	}
	if err != nil {
		return pain.Pain{}, err
	}
	return painFromPublished(row), nil
}

// ListLatest returns the newest published pains (FR-FEED-1).
func (r *Pains) ListLatest(ctx context.Context, limit, offset int) ([]pain.Pain, error) {
	rows, err := queries.New(handle(ctx, r.pool)).ListPublishedLatest(ctx, queries.ListPublishedLatestParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	return mapPublished(rows, err)
}

// ListTopVotes returns published pains ranked by vote count (FR-FEED-1).
func (r *Pains) ListTopVotes(ctx context.Context, limit, offset int) ([]pain.Pain, error) {
	rows, err := queries.New(handle(ctx, r.pool)).ListPublishedTopVotes(ctx, queries.ListPublishedTopVotesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	return mapPublished(rows, err)
}

// ListTrendingCandidates returns recent published pains with their 7-day vote
// window; the hot rank is finished by the pure scoring package in Go.
func (r *Pains) ListTrendingCandidates(ctx context.Context, limit int) ([]pain.Trending, error) {
	rows, err := queries.New(handle(ctx, r.pool)).ListPublishedTrendingCandidates(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]pain.Trending, 0, len(rows))
	for _, row := range rows {
		out = append(out, pain.Trending{
			Pain:    painFromTrendingRow(row),
			Votes7d: int(row.Votes7d),
		})
	}
	return out, nil
}

// ListByCategory returns a category's published pains (FR-HUB-1).
func (r *Pains) ListByCategory(ctx context.Context, category string, limit, offset int) ([]pain.Pain, error) {
	rows, err := queries.New(handle(ctx, r.pool)).ListPublishedByCategory(ctx, queries.ListPublishedByCategoryParams{
		Category: category,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	return mapPublished(rows, err)
}

// ListByCountry returns a country's published pains (FR-HUB-1).
func (r *Pains) ListByCountry(ctx context.Context, country string, limit, offset int) ([]pain.Pain, error) {
	rows, err := queries.New(handle(ctx, r.pool)).ListPublishedByCountry(ctx, queries.ListPublishedByCountryParams{
		Country: country,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	return mapPublished(rows, err)
}

// ListRelated returns other published pains in the same category, most urgent
// first (FR-DET-1).
func (r *Pains) ListRelated(ctx context.Context, category string, excludeID uuid.UUID, limit int) ([]pain.Pain, error) {
	rows, err := queries.New(handle(ctx, r.pool)).ListRelatedPublished(ctx, queries.ListRelatedPublishedParams{
		Category: category,
		ID:       excludeID,
		Limit:    int32(limit),
	})
	return mapPublished(rows, err)
}

// CountPublished is the total of published pains; hub indexability and SEO gate
// on it (FR-SEO-3).
func (r *Pains) CountPublished(ctx context.Context) (int, error) {
	n, err := queries.New(handle(ctx, r.pool)).CountPublishedTotal(ctx)
	return int(n), err
}

// CountByCategory counts a category's published pains.
func (r *Pains) CountByCategory(ctx context.Context, category string) (int, error) {
	n, err := queries.New(handle(ctx, r.pool)).CountPublishedByCategory(ctx, category)
	return int(n), err
}

// CountByCountry counts a country's published pains.
func (r *Pains) CountByCountry(ctx context.Context, country string) (int, error) {
	n, err := queries.New(handle(ctx, r.pool)).CountPublishedByCountry(ctx, country)
	return int(n), err
}

// RecentFromSource reports whether this source submitted a non-seeded pain
// within the window (dedupe guard, FR-SUB-7/FR-MOD-1). hashes are the identity
// package's monthly candidate set.
func (r *Pains) RecentFromSource(ctx context.Context, hashes []string, within time.Duration) (bool, error) {
	return queries.New(handle(ctx, r.pool)).RecentSubmissionFromSource(ctx, queries.RecentSubmissionFromSourceParams{
		Column1: hashes,
		Column2: pgtype.Interval{Microseconds: within.Microseconds(), Valid: true},
	})
}
