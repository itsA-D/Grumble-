package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"grumble/internal/store/postgres/queries"
)

// Votes is the repository behind anonymous voting (FR-VOTE-1/2). A vote is a
// (pain, voter_key) pair; counting and scoring never live here.
type Votes struct {
	pool *pgxpool.Pool
}

// NewVotes builds the vote repository on a pool.
func NewVotes(pool *pgxpool.Pool) *Votes {
	return &Votes{pool: pool}
}

// Add records a vote unless the key already voted on this pain. added reports
// whether a new row was created (repeat votes are a no-op). Useful inside a
// unit of work with Pains.UpdateCounters.
func (r *Votes) Add(ctx context.Context, painID uuid.UUID, voterKey string) (added bool, err error) {
	affected, err := queries.New(handle(ctx, r.pool)).InsertVote(ctx, queries.InsertVoteParams{
		PainID:   painID,
		VoterKey: voterKey,
	})
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// Remove deletes a vote. removed reports whether a row was actually deleted.
func (r *Votes) Remove(ctx context.Context, painID uuid.UUID, voterKey string) (removed bool, err error) {
	affected, err := queries.New(handle(ctx, r.pool)).DeleteVote(ctx, queries.DeleteVoteParams{
		PainID:   painID,
		VoterKey: voterKey,
	})
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// HasVoted reports whether the key already voted on this pain.
func (r *Votes) HasVoted(ctx context.Context, painID uuid.UUID, voterKey string) (bool, error) {
	return queries.New(handle(ctx, r.pool)).HasVoted(ctx, queries.HasVotedParams{
		PainID:   painID,
		VoterKey: voterKey,
	})
}

// Count returns the authoritative number of votes for a pain (FR-VOTE-2). The
// counter on pains is a cached copy updated in the same transaction.
func (r *Votes) Count(ctx context.Context, painID uuid.UUID) (int, error) {
	n, err := queries.New(handle(ctx, r.pool)).CountVotesForPain(ctx, painID)
	return int(n), err
}
