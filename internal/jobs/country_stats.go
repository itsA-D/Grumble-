package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"grumble/internal/store/postgres/queries"
)

// CountryStatsJobArgs carries no input: the job always rebuilds the whole read
// model, so a run is a pure function of the currently published pains.
type CountryStatsJobArgs struct{}

// Kind implements river.JobArgs.
func (CountryStatsJobArgs) Kind() string { return "country-stats" }

// CountryStatsJob recomputes the country_stats read model from published pains
// (FR-MAP-1). It is idempotent: it replaces the table contents in one
// transaction, so running it twice produces the same rows and stale countries
// disappear when their last pain is unpublished.
type CountryStatsJob struct {
	river.WorkerDefaults[CountryStatsJobArgs]
	pool *pgxpool.Pool
}

// NewCountryStatsJob builds the worker on the shared pool.
func NewCountryStatsJob(pool *pgxpool.Pool) *CountryStatsJob {
	return &CountryStatsJob{pool: pool}
}

// Timeout implements river.Worker.
func (j *CountryStatsJob) Timeout(*river.Job[CountryStatsJobArgs]) time.Duration {
	return 5 * time.Minute
}

// Work rebuilds country_stats from scratch inside one transaction.
func (j *CountryStatsJob) Work(ctx context.Context, _ *river.Job[CountryStatsJobArgs]) error {
	return RebuildCountryStats(ctx, j.pool)
}

// RebuildCountryStats replaces the whole read model in one transaction. It is
// shared by the periodic worker and the seed command so a fresh database has a
// populated map without waiting for the first scheduled run.
func RebuildCountryStats(ctx context.Context, pool *pgxpool.Pool) error {
	agg := &countryStatsAggregator{pool: pool}

	allRows, err := agg.aggregate(ctx, true)
	if err != nil {
		return fmt.Errorf("aggregate all categories: %w", err)
	}
	byCategory, err := agg.aggregate(ctx, false)
	if err != nil {
		return fmt.Errorf("aggregate by category: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "DELETE FROM country_stats"); err != nil {
		return fmt.Errorf("clear read model: %w", err)
	}

	q := queries.New(tx)
	for _, row := range append(allRows, byCategory...) {
		if err := q.UpsertCountryStats(ctx, queries.UpsertCountryStatsParams{
			Country:    row.country,
			Category:   row.category,
			PainCount:  row.painCount,
			VoteCount:  row.voteCount,
			AvgUrgency: row.avgUrgency,
		}); err != nil {
			return fmt.Errorf("upsert %s/%q: %w", row.country, row.category, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// countryStatRow is one computed aggregate row.
type countryStatRow struct {
	country    string
	category   string
	painCount  int32
	voteCount  int32
	avgUrgency int32
}

// countryStatsAggregator runs the read-only aggregation queries.
type countryStatsAggregator struct {
	pool *pgxpool.Pool
}

// aggregate computes either the all-category rows or the per-category rows.
// It reads the published-only view, so unpublished pains never leak (ADR 0001).
func (a *countryStatsAggregator) aggregate(ctx context.Context, allCategories bool) ([]countryStatRow, error) {
	var (
		query string
		rows  []countryStatRow
	)

	if allCategories {
		query = `
			SELECT country,
			       count(*)::int4,
			       coalesce(sum(vote_count), 0)::int4,
			       coalesce(round(avg(urgency)), 0)::int4
			FROM published_pains
			GROUP BY country`
	} else {
		query = `
			SELECT country, category,
			       count(*)::int4,
			       coalesce(sum(vote_count), 0)::int4,
			       coalesce(round(avg(urgency)), 0)::int4
			FROM published_pains
			GROUP BY country, category`
	}

	result, err := a.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var row countryStatRow
		if allCategories {
			if err := result.Scan(&row.country, &row.painCount, &row.voteCount, &row.avgUrgency); err != nil {
				return nil, err
			}
		} else {
			if err := result.Scan(&row.country, &row.category, &row.painCount, &row.voteCount, &row.avgUrgency); err != nil {
				return nil, err
			}
		}
		rows = append(rows, row)
	}
	return rows, result.Err()
}

// CountryStatsPeriodicJob returns the schedule that keeps the read model warm.
// Running every ten minutes is plenty for a board of this size and keeps the
// map current without aggregating on the request path.
func CountryStatsPeriodicJob() *river.PeriodicJob {
	return river.NewPeriodicJob(
		river.PeriodicInterval(10*time.Minute),
		func() (river.JobArgs, *river.InsertOpts) {
			return CountryStatsJobArgs{}, nil
		},
		&river.PeriodicJobOpts{RunOnStart: true},
	)
}
