//go:build integration

// Package postgres_test exercises the repositories against a real Postgres in a
// testcontainer. It covers the invariants the rest of M1 relies on: vote
// uniqueness (even under concurrency), published-only reads (unpublished rows
// never leak) and counters updated inside the change's transaction (PRD testing
// expectations).
package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"grumble/internal/domain"
	"grumble/internal/domain/pain"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/scoring"
	"grumble/internal/store/postgres"
)

type fixture struct {
	pool   *pgxpool.Pool
	pains  *postgres.Pains
	votes  *postgres.Votes
	tx     *postgres.TxManager
	source string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("grumble"),
		tcpostgres.WithUsername("grumble"),
		tcpostgres.WithPassword("grumble"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	if err := postgres.Migrate(ctx, url); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return &fixture{
		pool:   pool,
		pains:  postgres.NewPains(pool),
		votes:  postgres.NewVotes(pool),
		tx:     postgres.NewTxManager(pool),
		source: "source-hash-0001",
	}
}

func (f *fixture) seedPain(t *testing.T, slug string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	cat, _ := taxonomy.CategoryBySlug("housing")
	country, _ := taxonomy.CountryByCode("GB")
	_, err := f.pains.Create(context.Background(), pain.New{
		ID:       id,
		Slug:     slug,
		Body:     "A perfectly ordinary frustration that clears the minimum length.",
		Category: cat,
		Country:  country,
		Details: pain.Details{
			Frequency:     pain.FrequencyDaily,
			Disruption:    pain.DisruptionAnnoying,
			WouldPay:      true,
			MonthlyAmount: 10,
		},
		SourceHash:     f.source,
		ScoringVersion: scoring.Version,
	})
	if err != nil {
		t.Fatalf("create pain: %v", err)
	}
	return id
}

// castVote replicates the voting service's unit of work at the store level: lock
// the pain row, add the vote, recount and write the counters in one transaction.
func (f *fixture) castVote(ctx context.Context, id uuid.UUID, key string) error {
	return f.tx.InTx(ctx, func(ctx context.Context) error {
		vc, err := f.pains.LockForVote(ctx, id)
		if err != nil {
			return err
		}
		if _, err := f.votes.Add(ctx, id, key); err != nil {
			return err
		}
		count, err := f.votes.Count(ctx, id)
		if err != nil {
			return err
		}
		urgency := scoring.Urgency(scoring.Signals{
			Frequency:     string(vc.Frequency),
			Disruption:    string(vc.Disruption),
			WouldPay:      vc.WouldPay,
			MonthlyAmount: vc.MonthlyAmount,
			Votes:         count,
		})
		return f.pains.UpdateCounters(ctx, id, count, urgency)
	})
}

func (f *fixture) publish(t *testing.T, id uuid.UUID) {
	t.Helper()
	if err := f.pains.SetStatus(context.Background(), id, pain.StatusPublished); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

func TestUnpublishedRowsNeverLeak(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id := f.seedPain(t, "leaky-heating")

	if _, err := f.pains.PublishedBySlug(ctx, "leaky-heating"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("pending pain readable: err = %v, want ErrNotFound", err)
	}
	rows, err := f.pains.ListLatest(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list latest: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("pending pain appeared in the feed: %d rows", len(rows))
	}
	if n, _ := f.pains.CountPublished(ctx); n != 0 {
		t.Fatalf("published count = %d, want 0", n)
	}

	f.publish(t, id)
	if _, err := f.pains.PublishedBySlug(ctx, "leaky-heating"); err != nil {
		t.Fatalf("published pain not readable: %v", err)
	}
	if n, _ := f.pains.CountPublished(ctx); n != 1 {
		t.Fatalf("published count = %d, want 1", n)
	}
}

func TestVoteUniquenessAndToggle(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id := f.seedPain(t, "cold-coffee")
	f.publish(t, id)
	key := "a:anon-1"

	if added, err := f.votes.Add(ctx, id, key); err != nil || !added {
		t.Fatalf("first vote: added=%v err=%v", added, err)
	}
	if added, err := f.votes.Add(ctx, id, key); err != nil || added {
		t.Fatalf("repeat vote must be a no-op: added=%v err=%v", added, err)
	}
	if voted, _ := f.votes.HasVoted(ctx, id, key); !voted {
		t.Fatal("HasVoted = false after voting")
	}
	if removed, err := f.votes.Remove(ctx, id, key); err != nil || !removed {
		t.Fatalf("remove: removed=%v err=%v", removed, err)
	}
	if voted, _ := f.votes.HasVoted(ctx, id, key); voted {
		t.Fatal("HasVoted = true after removal")
	}
	if removed, _ := f.votes.Remove(ctx, id, key); removed {
		t.Fatal("second remove must report no change")
	}
}

// TestConcurrentVotesSameSourceAreSerialised is the concurrency guarantee the
// voting service relies on: twenty parallel votes from one source leave exactly
// one vote row and a counter of one, never a lost update.
func TestConcurrentVotesSameSourceAreSerialised(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id := f.seedPain(t, "concurrent-bus")
	f.publish(t, id)

	const voters = 20
	var wg sync.WaitGroup
	errs := make(chan error, voters)
	wg.Add(voters)
	for i := 0; i < voters; i++ {
		go func() {
			defer wg.Done()
			if err := f.castVote(ctx, id, "a:same-source"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent vote: %v", err)
	}

	var rows, voteCount int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM votes WHERE pain_id = $1", id).Scan(&rows); err != nil {
		t.Fatalf("count votes: %v", err)
	}
	if err := f.pool.QueryRow(ctx, "SELECT vote_count FROM pains WHERE id = $1", id).Scan(&voteCount); err != nil {
		t.Fatalf("read counter: %v", err)
	}
	if rows != 1 {
		t.Fatalf("vote rows = %d, want 1 (one voter, one row)", rows)
	}
	if voteCount != 1 {
		t.Fatalf("vote_count = %d, want 1 (no lost update)", voteCount)
	}
}

func TestCountersUpdateInsideTransaction(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id := f.seedPain(t, "slow-internet")
	f.publish(t, id)

	err := f.tx.InTx(ctx, func(ctx context.Context) error {
		added, err := f.votes.Add(ctx, id, "a:anon-2")
		if err != nil {
			return err
		}
		if !added {
			return errors.New("expected a new vote")
		}
		urgency := scoring.Urgency(scoring.Signals{
			Frequency: "daily", Disruption: "annoying", WouldPay: true, MonthlyAmount: 10, Votes: 1,
		})
		return f.pains.UpdateCounters(ctx, id, 1, urgency)
	})
	if err != nil {
		t.Fatalf("unit of work: %v", err)
	}

	var voteCount, urgency int
	if err := f.pool.QueryRow(ctx,
		"SELECT vote_count, urgency FROM pains WHERE id = $1", id,
	).Scan(&voteCount, &urgency); err != nil {
		t.Fatalf("read counters: %v", err)
	}
	if voteCount != 1 {
		t.Fatalf("vote_count = %d, want 1", voteCount)
	}
	if urgency <= 0 {
		t.Fatalf("urgency = %d, want > 0", urgency)
	}
}

func TestTransactionRollbackLeavesNoTrace(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id := f.seedPain(t, "vanishing-act")
	before, _ := f.pains.CountPublished(ctx)

	sentinel := errors.New("boom")
	err := f.tx.InTx(ctx, func(ctx context.Context) error {
		if err := f.pains.SetStatus(ctx, id, pain.StatusPublished); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("InTx error = %v, want sentinel", err)
	}
	if _, err := f.pains.PublishedBySlug(ctx, "vanishing-act"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("rollback did not happen: err = %v", err)
	}
	if after, _ := f.pains.CountPublished(ctx); after != before {
		t.Fatalf("published count changed across rollback: %d -> %d", before, after)
	}
}

func TestRecentFromSource(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.seedPain(t, "repeat-offender")

	found, err := f.pains.RecentFromSource(ctx, []string{f.source}, 24*time.Hour)
	if err != nil {
		t.Fatalf("recent from source: %v", err)
	}
	if !found {
		t.Fatal("expected the just-created submission to match")
	}
	other, err := f.pains.RecentFromSource(ctx, []string{"different-hash"}, 24*time.Hour)
	if err != nil {
		t.Fatalf("recent from source: %v", err)
	}
	if other {
		t.Fatal("a different source hash must not match")
	}
}
