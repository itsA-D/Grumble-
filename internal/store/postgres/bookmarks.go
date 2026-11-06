package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"grumble/internal/domain/pain"
	"grumble/internal/store/postgres/queries"
)

// Bookmarks is the private saved-list repository (PRD FR-BKM-1). Public reads
// join published_pains; the owner key is never exposed outside the service.
type Bookmarks struct {
	pool *pgxpool.Pool
}

// NewBookmarks builds the bookmark repository on a pool.
func NewBookmarks(pool *pgxpool.Pool) *Bookmarks {
	return &Bookmarks{pool: pool}
}

// Add saves a published pain. added is false for a repeat save or an
// unpublished/missing id.
func (r *Bookmarks) Add(ctx context.Context, ownerKey string, painID uuid.UUID) (bool, error) {
	n, err := queries.New(handle(ctx, r.pool)).InsertBookmark(ctx, queries.InsertBookmarkParams{
		OwnerKey: ownerKey,
		PainID:   painID,
	})
	return n == 1, err
}

// Remove deletes a saved row. It is idempotent.
func (r *Bookmarks) Remove(ctx context.Context, ownerKey string, painID uuid.UUID) error {
	_, err := queries.New(handle(ctx, r.pool)).DeleteBookmark(ctx, queries.DeleteBookmarkParams{
		OwnerKey: ownerKey,
		PainID:   painID,
	})
	return err
}

// Has reports whether the owner has saved the pain.
func (r *Bookmarks) Has(ctx context.Context, ownerKey string, painID uuid.UUID) (bool, error) {
	return queries.New(handle(ctx, r.pool)).HasBookmark(ctx, queries.HasBookmarkParams{
		OwnerKey: ownerKey,
		PainID:   painID,
	})
}

// Count returns how many rows the owner has saved.
func (r *Bookmarks) Count(ctx context.Context, ownerKey string) (int, error) {
	n, err := queries.New(handle(ctx, r.pool)).CountBookmarks(ctx, ownerKey)
	return int(n), err
}

// IDs returns the published, saved ids in newest-save order.
func (r *Bookmarks) IDs(ctx context.Context, ownerKey string) ([]uuid.UUID, error) {
	return queries.New(handle(ctx, r.pool)).ListBookmarkIDs(ctx, ownerKey)
}

// List returns the published saved pains in newest-save order.
func (r *Bookmarks) List(ctx context.Context, ownerKey string, limit int) ([]pain.Pain, error) {
	rows, err := queries.New(handle(ctx, r.pool)).ListBookmarks(ctx, queries.ListBookmarksParams{
		OwnerKey:  ownerKey,
		PageLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]pain.Pain, 0, len(rows))
	for _, row := range rows {
		out = append(out, painFromBookmarkRow(row))
	}
	return out, nil
}
