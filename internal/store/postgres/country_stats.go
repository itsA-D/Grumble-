package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"grumble/internal/domain/mapdata"
	"grumble/internal/store/postgres/queries"
)

// CountryStats is the read-model repository for the world map. It only reads
// the precomputed country_stats table; writes are done by the periodic job.
type CountryStats struct {
	pool *pgxpool.Pool
}

// NewCountryStats builds the country stats repository on a pool.
func NewCountryStats(pool *pgxpool.Pool) *CountryStats {
	return &CountryStats{pool: pool}
}

// List returns all countries for a given category (empty = all categories).
func (r *CountryStats) List(ctx context.Context, category string) ([]mapdata.StatRow, error) {
	q := queries.New(handle(ctx, r.pool))
	rows, err := q.ListCountryStats(ctx, category)
	if err != nil {
		return nil, err
	}
	out := make([]mapdata.StatRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapdata.StatRow{
			Country:    row.Country,
			PainCount:  int(row.PainCount),
			VoteCount:  int(row.VoteCount),
			AvgUrgency: int(row.AvgUrgency),
		})
	}
	return out, nil
}

// ListTotals returns totals per country across all categories.
func (r *CountryStats) ListTotals(ctx context.Context) ([]mapdata.StatRow, error) {
	q := queries.New(handle(ctx, r.pool))
	rows, err := q.ListCountryStatsTotals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]mapdata.StatRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapdata.StatRow{
			Country:    row.Country,
			PainCount:  int(row.PainCount),
			VoteCount:  int(row.VoteCount),
			AvgUrgency: int(row.AvgUrgency),
		})
	}
	return out, nil
}

// CountCountriesWithPains counts distinct countries with at least one pain.
func (r *CountryStats) CountCountriesWithPains(ctx context.Context) (int, error) {
	q := queries.New(handle(ctx, r.pool))
	n, err := q.CountCountriesWithPains(ctx)
	return int(n), err
}

// SumPainCount returns the total pain count across all countries.
func (r *CountryStats) SumPainCount(ctx context.Context) (int, error) {
	q := queries.New(handle(ctx, r.pool))
	n, err := q.SumPainCount(ctx)
	return int(n), err
}

// SumVoteCount returns the total vote count across all countries.
func (r *CountryStats) SumVoteCount(ctx context.Context) (int, error) {
	q := queries.New(handle(ctx, r.pool))
	n, err := q.SumVoteCount(ctx)
	return int(n), err
}

// GetTopCountries returns the top N countries by pain count.
func (r *CountryStats) GetTopCountries(ctx context.Context, limit int) ([]mapdata.CountryRow, error) {
	q := queries.New(handle(ctx, r.pool))
	rows, err := q.GetTopCountries(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]mapdata.CountryRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapdata.CountryRow{Country: row.Country, PainCount: int(row.PainCount)})
	}
	return out, nil
}

// GetTopCountriesByCategory returns the top N countries for a category.
func (r *CountryStats) GetTopCountriesByCategory(ctx context.Context, category string, limit int) ([]mapdata.CountryRow, error) {
	q := queries.New(handle(ctx, r.pool))
	rows, err := q.GetTopCountriesByCategory(ctx, queries.GetTopCountriesByCategoryParams{
		Category: category,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]mapdata.CountryRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapdata.CountryRow{Country: row.Country, PainCount: int(row.PainCount)})
	}
	return out, nil
}

// ListTopPainsByCountry returns top N published pains for a country.
func (r *CountryStats) ListTopPainsByCountry(ctx context.Context, country, category string, limit int) ([]mapdata.PainRow, error) {
	q := queries.New(handle(ctx, r.pool))
	rows, err := q.ListTopPainsByCountry(ctx, queries.ListTopPainsByCountryParams{
		Country: country,
		Column2: category,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]mapdata.PainRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapdata.PainRow{
			Slug:      row.Slug,
			Body:      row.Body,
			Category:  row.Category,
			Country:   row.Country,
			VoteCount: int(row.VoteCount),
			Urgency:   int(row.Urgency),
			Seeded:    row.Seeded,
		})
	}
	return out, nil
}

// Upsert writes a country+category stats row (called by the periodic job).
func (r *CountryStats) Upsert(ctx context.Context, country, category string, painCount, voteCount, avgUrgency int) error {
	q := queries.New(handle(ctx, r.pool))
	return q.UpsertCountryStats(ctx, queries.UpsertCountryStatsParams{
		Country:    country,
		Category:   category,
		PainCount:  int32(painCount),
		VoteCount:  int32(voteCount),
		AvgUrgency: int32(avgUrgency),
	})
}
