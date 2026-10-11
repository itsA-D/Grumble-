// Package voting toggles a pain's vote and recomputes its urgency and counter
// in the same transaction (PRD FR-VOTE-1, FR-VOTE-2; ARCHITECTURE.md invariants
// 6 and the Transfers/DTO boundary). A vote row is unique per (pain, voter), so
// the operation is lock-serialised and safe under concurrency.
package voting

import (
	"context"

	"github.com/google/uuid"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
	"grumble/internal/scoring"
)

// Store is the pain persistence the service needs. LockForVote must lock the
// pain row for the duration of the transaction.
type Store interface {
	LockForVote(ctx context.Context, id uuid.UUID) (pain.VoteContext, error)
	UpdateCounters(ctx context.Context, id uuid.UUID, voteCount, urgency int) error
}

// Votes is the vote persistence the service needs.
type Votes interface {
	HasVoted(ctx context.Context, painID uuid.UUID, voterKey string) (bool, error)
	Add(ctx context.Context, painID uuid.UUID, voterKey string) (bool, error)
	Remove(ctx context.Context, painID uuid.UUID, voterKey string) (bool, error)
	Count(ctx context.Context, painID uuid.UUID) (int, error)
}

// TxRunner runs a unit of work in one transaction.
type TxRunner interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service changes votes.
type Service struct {
	store Store
	votes Votes
	tx    TxRunner
}

// New builds the voting service.
func New(store Store, votes Votes, tx TxRunner) *Service {
	return &Service{store: store, votes: votes, tx: tx}
}

// Request identifies the pain and the voter, and states the desired end state.
type Request struct {
	PainID   uuid.UUID
	VoterKey string
	// Voted is the desired state: true to add a vote, false to remove one.
	Voted bool
}

// Result is the pain's state after the change, echoed to the client so the UI
// can confirm or correct an optimistic update.
type Result struct {
	Voted     bool
	VoteCount int
	Urgency   int
}

// Set brings the vote to the requested state (idempotent): it adds a vote when
// absent, removes it when present, and never double-counts a repeat. The UI
// renders this as a toggle by sending the opposite of the current state
// (FR-VOTE-1).
func (s *Service) Set(ctx context.Context, req Request) (Result, error) {
	var res Result
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		vc, err := s.lockPublished(ctx, req.PainID)
		if err != nil {
			return err
		}
		has, err := s.votes.HasVoted(ctx, req.PainID, req.VoterKey)
		if err != nil {
			return err
		}
		if req.Voted && !has {
			if _, err := s.votes.Add(ctx, req.PainID, req.VoterKey); err != nil {
				return err
			}
		} else if !req.Voted && has {
			if _, err := s.votes.Remove(ctx, req.PainID, req.VoterKey); err != nil {
				return err
			}
		}
		res, err = s.finish(ctx, vc, req.Voted)
		return err
	})
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

// Toggle flips the caller's vote and reports the new state. It reads the
// current state under the same row lock, so two concurrent toggles from one key
// serialise rather than race.
func (s *Service) Toggle(ctx context.Context, painID uuid.UUID, voterKey string) (Result, error) {
	var res Result
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		vc, err := s.lockPublished(ctx, painID)
		if err != nil {
			return err
		}
		has, err := s.votes.HasVoted(ctx, painID, voterKey)
		if err != nil {
			return err
		}
		want := !has
		if want {
			_, err = s.votes.Add(ctx, painID, voterKey)
		} else {
			_, err = s.votes.Remove(ctx, painID, voterKey)
		}
		if err != nil {
			return err
		}
		res, err = s.finish(ctx, vc, want)
		return err
	})
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

// Voted reports whether the identity has already voted on the pain. It is a
// plain read used to render the toggle's current state (FR-VOTE-1).
func (s *Service) Voted(ctx context.Context, painID uuid.UUID, voterKey string) (bool, error) {
	return s.votes.HasVoted(ctx, painID, voterKey)
}

// lockPublished locks the pain row and rejects rows that are not publicly
// visible.
func (s *Service) lockPublished(ctx context.Context, id uuid.UUID) (pain.VoteContext, error) {
	vc, err := s.store.LockForVote(ctx, id)
	if err != nil {
		return pain.VoteContext{}, err
	}
	if !vc.Status.Public() {
		return pain.VoteContext{}, domain.ErrNotFound
	}
	return vc, nil
}

// finish counts the votes, recomputes urgency through the pure scoring package
// and writes both counters on the already-locked row (invariant 6).
func (s *Service) finish(ctx context.Context, vc pain.VoteContext, voted bool) (Result, error) {
	count, err := s.votes.Count(ctx, vc.ID)
	if err != nil {
		return Result{}, err
	}
	urgency := scoring.Urgency(signalsFor(vc, count))
	if err := s.store.UpdateCounters(ctx, vc.ID, count, urgency); err != nil {
		return Result{}, err
	}
	return Result{Voted: voted, VoteCount: count, Urgency: urgency}, nil
}

// signalsFor projects a locked pain row plus the current vote count into the
// pure scoring input (PRD section 9.1).
func signalsFor(c pain.VoteContext, votes int) scoring.Signals {
	return scoring.Signals{
		Frequency:     string(c.Frequency),
		Disruption:    string(c.Disruption),
		WouldPay:      c.WouldPay,
		MonthlyAmount: c.MonthlyAmount,
		Votes:         votes,
	}
}
