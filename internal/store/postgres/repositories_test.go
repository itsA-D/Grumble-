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
	"grumble/internal/jobs"
	"grumble/internal/scoring"
	"grumble/internal/store/postgres"
)

type fixture struct {
	pool      *pgxpool.Pool
	pains     *postgres.Pains
	votes     *postgres.Votes
	bookmarks *postgres.Bookmarks
	tx        *postgres.TxManager
	source    string
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
		pool:      pool,
		pains:     postgres.NewPains(pool),
		votes:     postgres.NewVotes(pool),
		bookmarks: postgres.NewBookmarks(pool),
		tx:        postgres.NewTxManager(pool),
		source:    "source-hash-0001",
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

func TestPublishedStatsCountOnlyPublishedRows(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id := f.seedPain(t, "countable-heating")
	f.publish(t, id)
	if err := f.castVote(ctx, id, "a:stat-voter"); err != nil {
		t.Fatalf("cast vote: %v", err)
	}
	_ = f.seedPain(t, "pending-heating")

	pains, err := f.pains.CountPublished(ctx)
	if err != nil {
		t.Fatalf("count published: %v", err)
	}
	countries, err := f.pains.CountCountries(ctx)
	if err != nil {
		t.Fatalf("count countries: %v", err)
	}
	votes, err := f.pains.CountVotes(ctx)
	if err != nil {
		t.Fatalf("count votes: %v", err)
	}
	if pains != 1 || countries != 1 || votes != 1 {
		t.Fatalf("stats = %d pains / %d countries / %d votes, want 1/1/1", pains, countries, votes)
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

// TestCountryStatsRebuildIsIdempotentAndPublishedOnly covers FR-MAP-1: the read
// model is rebuilt from published pains only, and running the job twice leaves
// the same rows instead of duplicating or drifting.
func TestCountryStatsRebuildIsIdempotentAndPublishedOnly(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	published := f.seedPain(t, "published-in-gb")
	f.publish(t, published)

	// An unpublished pain in another country must never reach the read model.
	unpublishedID := uuid.New()
	tech, _ := taxonomy.CategoryBySlug("tech")
	fr, _ := taxonomy.CountryByCode("FR")
	if _, err := f.pains.Create(ctx, pain.New{
		ID:       unpublishedID,
		Slug:     "held-in-fr",
		Body:     "A perfectly ordinary frustration that clears the minimum length.",
		Category: tech,
		Country:  fr,
		Details: pain.Details{
			Frequency:     pain.FrequencyDaily,
			Disruption:    pain.DisruptionAnnoying,
			WouldPay:      false,
			MonthlyAmount: 0,
		},
		SourceHash:     "source-hash-0002",
		ScoringVersion: scoring.Version,
	}); err != nil {
		t.Fatalf("create unpublished pain: %v", err)
	}

	if err := jobs.RebuildCountryStats(ctx, f.pool); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	var allGB, housingGB int
	if err := f.pool.QueryRow(ctx, `SELECT pain_count FROM country_stats WHERE country = 'GB' AND category = ''`).Scan(&allGB); err != nil {
		t.Fatalf("read GB total: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT pain_count FROM country_stats WHERE country = 'GB' AND category = 'housing'`).Scan(&housingGB); err != nil {
		t.Fatalf("read GB housing: %v", err)
	}
	if allGB != 1 || housingGB != 1 {
		t.Fatalf("GB counts = all:%d housing:%d, want 1 and 1", allGB, housingGB)
	}

	var frRows int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM country_stats WHERE country = 'FR'`).Scan(&frRows); err != nil {
		t.Fatalf("read FR rows: %v", err)
	}
	if frRows != 0 {
		t.Fatalf("unpublished FR pain leaked into the read model: %d rows", frRows)
	}

	var before int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM country_stats`).Scan(&before); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if err := jobs.RebuildCountryStats(ctx, f.pool); err != nil {
		t.Fatalf("second rebuild: %v", err)
	}
	var after int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM country_stats`).Scan(&after); err != nil {
		t.Fatalf("count rows again: %v", err)
	}
	if before != after {
		t.Fatalf("rebuild is not idempotent: %d rows then %d", before, after)
	}
}

func TestBookmarksArePrivatePublishedAndIdempotent(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	first := f.seedPain(t, "saved-heating")
	second := f.seedPain(t, "saved-buses")
	pending := f.seedPain(t, "saved-pending")
	f.publish(t, first)
	f.publish(t, second)

	const owner = "a:bookmark-owner"
	const other = "a:other-owner"

	added, err := f.bookmarks.Add(ctx, owner, first)
	if err != nil || !added {
		t.Fatalf("first add = %v, %v", added, err)
	}
	added, err = f.bookmarks.Add(ctx, owner, first)
	if err != nil || added {
		t.Fatalf("repeat add = %v, %v, want false", added, err)
	}
	if added, err := f.bookmarks.Add(ctx, owner, pending); err != nil || added {
		t.Fatalf("pending add = %v, %v, want false", added, err)
	}

	time.Sleep(2 * time.Millisecond)
	if added, err := f.bookmarks.Add(ctx, owner, second); err != nil || !added {
		t.Fatalf("second saved pain: %v, %v", added, err)
	}

	list, err := f.bookmarks.List(ctx, owner, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0].ID != second {
		t.Fatalf("list = %+v, want second newest first", list)
	}
	if n, err := f.bookmarks.Count(ctx, owner); err != nil || n != 2 {
		t.Fatalf("count = %d, %v, want 2", n, err)
	}
	if list, err := f.bookmarks.List(ctx, other, 10); err != nil || len(list) != 0 {
		t.Fatalf("other owner sees %d rows, %v", len(list), err)
	}

	// Unpublishing removes the row from every read and from the cap count, but
	// never reveals it.
	if err := f.pains.SetStatus(ctx, first, pain.StatusRejected); err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	ids, err := f.bookmarks.IDs(ctx, owner)
	if err != nil || len(ids) != 1 || ids[0] != second {
		t.Fatalf("ids after unpublish = %v, %v", ids, err)
	}
	if n, err := f.bookmarks.Count(ctx, owner); err != nil || n != 1 {
		t.Fatalf("count after unpublish = %d, %v, want 1", n, err)
	}

	if err := f.bookmarks.Remove(ctx, owner, second); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if n, err := f.bookmarks.Count(ctx, owner); err != nil || n != 0 {
		t.Fatalf("count after remove = %d, %v", n, err)
	}
}

func TestTrendingWindowOnlyReadsPublishedRows(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	published := f.seedPain(t, "trending-heating")
	f.publish(t, published)
	f.seedPain(t, "trending-pending")

	rows, err := f.pains.ListTrendingWindow(ctx, time.Now().Add(-24*time.Hour), 50)
	if err != nil {
		t.Fatalf("list trending: %v", err)
	}
	if len(rows) != 1 || rows[0].Pain.ID != published {
		t.Fatalf("rows = %+v, want only the published pain", rows)
	}
	if rows[0].Votes7d != 0 {
		t.Fatalf("votes window = %d, want 0", rows[0].Votes7d)
	}

	if err := f.castVote(ctx, published, "a:trend-voter"); err != nil {
		t.Fatalf("cast vote: %v", err)
	}
	rows, err = f.pains.ListTrendingWindow(ctx, time.Now().Add(-24*time.Hour), 50)
	if err != nil {
		t.Fatalf("list trending after vote: %v", err)
	}
	if len(rows) != 1 || rows[0].Votes7d != 1 {
		t.Fatalf("votes window after vote = %+v, want 1", rows)
	}

	cats, err := f.pains.ListTrendingCategoryCounts(ctx, time.Now().Add(-24*time.Hour), 8)
	if err != nil || len(cats) != 1 || cats[0].Count != 1 {
		t.Fatalf("categories = %+v, %v", cats, err)
	}
}
