// Package bookmarks owns anonymous saved frustrations (PRD FR-BKM-1). Saves
// are idempotent, private to the anonymous identity, and capped; the repository
// enforces published-only reads.
package bookmarks

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
)

// ErrLimit means the identity has reached its saved-list cap.
var ErrLimit = errors.New("bookmarks: saved-list limit reached")

// Store is the bookmark persistence the service needs.
type Store interface {
	Add(ctx context.Context, ownerKey string, painID uuid.UUID) (bool, error)
	Remove(ctx context.Context, ownerKey string, painID uuid.UUID) error
	Has(ctx context.Context, ownerKey string, painID uuid.UUID) (bool, error)
	Count(ctx context.Context, ownerKey string) (int, error)
	IDs(ctx context.Context, ownerKey string) ([]uuid.UUID, error)
	List(ctx context.Context, ownerKey string, limit int) ([]pain.Pain, error)
}

// TxRunner runs a unit of work in one transaction.
type TxRunner interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Config bounds the list.
type Config struct {
	// SavedLimit is the maximum number of saved frustrations.
	SavedLimit int
}

// Service reads and mutates one identity's saved list.
type Service struct {
	store Store
	tx    TxRunner
	cfg   Config
}

// New builds the bookmark service.
func New(store Store, tx TxRunner, cfg Config) *Service {
	if cfg.SavedLimit < 1 {
		cfg.SavedLimit = 5
	}
	return &Service{store: store, tx: tx, cfg: cfg}
}

// Limit is the configured saved-list cap.
func (s *Service) Limit() int { return s.cfg.SavedLimit }

// Result is the state after a save or remove.
type Result struct {
	Saved bool
	Count int
	// Limit is duplicated so the handler can render the cap without config.
	Limit int
}

// Save adds a published pain unless it is already saved. It returns ErrLimit
// when the cap is reached and domain.ErrNotFound for an unpublished id.
func (s *Service) Save(ctx context.Context, ownerKey string, painID uuid.UUID) (Result, error) {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return Result{}, domain.ErrNotFound
	}

	var res Result
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		has, err := s.store.Has(ctx, ownerKey, painID)
		if err != nil {
			return err
		}
		if has {
			count, err := s.store.Count(ctx, ownerKey)
			if err != nil {
				return err
			}
			res = Result{Saved: true, Count: count, Limit: s.cfg.SavedLimit}
			return nil
		}
		count, err := s.store.Count(ctx, ownerKey)
		if err != nil {
			return err
		}
		if count >= s.cfg.SavedLimit {
			return ErrLimit
		}
		added, err := s.store.Add(ctx, ownerKey, painID)
		if err != nil {
			return err
		}
		if !added {
			// The SELECT inside Add filters unpublished pains; distinguish a
			// concurrent duplicate from an id that is not public.
			has, err := s.store.Has(ctx, ownerKey, painID)
			if err != nil {
				return err
			}
			if !has {
				return domain.ErrNotFound
			}
		}
		count, err = s.store.Count(ctx, ownerKey)
		if err != nil {
			return err
		}
		res = Result{Saved: true, Count: count, Limit: s.cfg.SavedLimit}
		return nil
	})
	return res, err
}

// Remove deletes a saved pain. It is idempotent and returns the new count.
func (s *Service) Remove(ctx context.Context, ownerKey string, painID uuid.UUID) (Result, error) {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return Result{}, domain.ErrNotFound
	}
	var res Result
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.store.Remove(ctx, ownerKey, painID); err != nil {
			return err
		}
		count, err := s.store.Count(ctx, ownerKey)
		if err != nil {
			return err
		}
		res = Result{Saved: false, Count: count, Limit: s.cfg.SavedLimit}
		return nil
	})
	return res, err
}

// List returns the saved page in newest-save order.
func (s *Service) List(ctx context.Context, ownerKey string, limit int) ([]pain.Pain, error) {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = s.cfg.SavedLimit
	}
	return s.store.List(ctx, ownerKey, limit)
}

// IDs returns the identity's saved ids as a set for server-rendered toggles.
func (s *Service) IDs(ctx context.Context, ownerKey string) (map[uuid.UUID]bool, error) {
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return map[uuid.UUID]bool{}, nil
	}
	ids, err := s.store.IDs(ctx, ownerKey)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
