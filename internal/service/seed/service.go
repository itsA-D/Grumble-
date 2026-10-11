// Package seed loads the labelled demo content described in the PRD. Every row
// it writes is flagged seeded = true and is stored already published; the public
// never sees it as organic. Seeding is idempotent: each pain is keyed by a
// deterministic slug and a rerun inserts nothing new (FR-SUB-8, PRD section 6).
package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"grumble/internal/domain/pain"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/scoring"
)

// sourceHashSeed is the non-personal placeholder stored as the source hash of a
// seed row. Seeds never come from a real client address (NFR-PRIV-1).
const sourceHashSeed = "seed"

// Store is the persistence the seeder needs. CreateSeeded must ignore a slug
// conflict so a rerun inserts nothing.
type Store interface {
	CreateSeeded(ctx context.Context, in pain.New) (id uuid.UUID, created bool, err error)
	AddSeedVotes(ctx context.Context, painID uuid.UUID, voterKeys []string) (int, error)
	UpdateCounters(ctx context.Context, id uuid.UUID, voteCount, urgency int) error
}

// TxRunner runs a unit of work in one transaction.
type TxRunner interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service loads the seed content.
type Service struct {
	store Store
	tx    TxRunner
	newID func() uuid.UUID
	items []Item
}

// New builds the seed service. newID mints primary keys (UUIDv7 in production).
func New(store Store, tx TxRunner, newID func() uuid.UUID, items []Item) *Service {
	return &Service{store: store, tx: tx, newID: newID, items: items}
}

// Report summarises a seed run.
type Report struct {
	Created int
	Skipped int
}

// Run inserts every seed item that is not already present. Each item is written
// in one transaction: the published numbered row, its synthetic votes, and the
// urgency computed from the real vote count (invariant 6).
func (s *Service) Run(ctx context.Context) (Report, error) {
	var rep Report
	for _, item := range s.items {
		created, err := s.insert(ctx, item)
		if err != nil {
			return rep, fmt.Errorf("seed %q: %w", item.Body, err)
		}
		if created {
			rep.Created++
		} else {
			rep.Skipped++
		}
	}
	return rep, nil
}

func (s *Service) insert(ctx context.Context, item Item) (bool, error) {
	category, err := resolveCategory(item.Category)
	if err != nil {
		return false, err
	}
	country, err := resolveCountry(item.Country)
	if err != nil {
		return false, err
	}
	in := pain.New{
		ID:       s.newID(),
		Slug:     pain.Slugify(item.Body),
		Body:     item.Body,
		Category: category,
		Country:  country,
		Details: pain.Details{
			Frequency:     item.Frequency,
			Disruption:    item.Disruption,
			WouldPay:      item.WouldPay,
			MonthlyAmount: item.MonthlyAmount,
			Tried:         item.Tried,
		},
		SourceHash:     sourceHashSeed,
		Seeded:         true,
		ScoringVersion: scoring.Version,
	}

	var created bool
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		id, ok, err := s.store.CreateSeeded(ctx, in)
		if err != nil || !ok {
			created = ok
			return err
		}
		n, err := s.store.AddSeedVotes(ctx, id, voterKeys(in.Slug, item.Votes))
		if err != nil {
			return err
		}
		urgency := scoring.Urgency(scoring.Signals{
			Frequency:     string(item.Frequency),
			Disruption:    string(item.Disruption),
			WouldPay:      item.WouldPay,
			MonthlyAmount: item.MonthlyAmount,
			Votes:         n,
		})
		created = true
		return s.store.UpdateCounters(ctx, id, n, urgency)
	})
	return created, err
}

// voterKeys builds stable synthetic voter keys so a re-run would insert the same
// set and conflict rather than double-count. Keys are never real identities.
func voterKeys(slug string, votes int) []string {
	if votes <= 0 {
		return nil
	}
	keys := make([]string, votes)
	for i := range keys {
		keys[i] = fmt.Sprintf("a:seed-%s-%d", slug, i)
	}
	return keys
}

func resolveCategory(slug string) (taxonomy.Category, error) {
	c, ok := taxonomy.CategoryBySlug(slug)
	if !ok {
		return taxonomy.Category{}, fmt.Errorf("unknown category %q", slug)
	}
	return c, nil
}

func resolveCountry(code string) (taxonomy.Country, error) {
	c, ok := taxonomy.CountryByCode(code)
	if !ok {
		return taxonomy.Country{}, fmt.Errorf("unknown country %q", code)
	}
	return c, nil
}

// DefaultItems returns the built-in labelled seed content.
func DefaultItems() []Item { return content }
