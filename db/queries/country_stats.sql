-- country_stats.sql
-- Read model queries for the world map (PRD FR-MAP-1). All reads are
-- from the precomputed country_stats table; no live aggregation on the
-- published_pains view. An empty category string means "all categories".

-- name: UpsertCountryStats :exec
-- Idempotent upsert for the periodic job. The primary key (country, category)
-- ensures running twice replaces the row with the same values.
INSERT INTO country_stats (country, category, pain_count, vote_count, avg_urgency, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (country, category) DO UPDATE SET
    pain_count  = EXCLUDED.pain_count,
    vote_count  = EXCLUDED.vote_count,
    avg_urgency = EXCLUDED.avg_urgency,
    updated_at  = now();

-- name: GetCountryStats :one
-- Single country+category row for the side panel.
SELECT country, category, pain_count, vote_count, avg_urgency
FROM country_stats
WHERE country = $1 AND category = $2;

-- name: ListCountryStats :many
-- All countries for a given category ('' = all categories). Ordered by
-- pain_count desc for the top-N list and the choropleth buckets.
SELECT country, pain_count, vote_count, avg_urgency
FROM country_stats
WHERE category = $1
ORDER BY pain_count DESC, country;

-- name: ListCountryStatsTotals :many
-- Totals per country across all categories (the '' aggregate rows).
SELECT country, pain_count, vote_count, avg_urgency
FROM country_stats_totals
ORDER BY pain_count DESC, country;

-- name: CountCountriesWithPains :one
-- Distinct countries that have at least one published pain (for the stat strip).
SELECT count(*) FROM country_stats_totals WHERE pain_count > 0;

-- name: SumPainCount :one
-- Total published pains across all countries (for the stat strip).
SELECT COALESCE(SUM(pain_count), 0)::bigint FROM country_stats_totals;

-- name: SumVoteCount :one
-- Total votes across all countries (for the stat strip).
SELECT COALESCE(SUM(vote_count), 0)::bigint FROM country_stats_totals;

-- name: GetTopCountries :many
-- Top N countries by pain_count for the side panel "TOP COUNTRIES" list.
SELECT country, pain_count
FROM country_stats_totals
WHERE pain_count > 0
ORDER BY pain_count DESC, country
LIMIT $1;

-- name: GetTopCountriesByCategory :many
-- Top N countries for a specific category.
SELECT country, pain_count
FROM country_stats
WHERE category = $1 AND pain_count > 0
ORDER BY pain_count DESC, country
LIMIT $2;

-- name: ListTopPainsByCountry :many
-- Top N published pains for a country, ordered by urgency desc, vote_count desc.
-- Used for the side panel when a country is selected.
SELECT id, slug, body, category, country, vote_count, urgency, seeded
FROM published_pains
WHERE country = $1
  AND ($2::text = '' OR category = $2::text)
ORDER BY urgency DESC, vote_count DESC, created_at DESC
LIMIT $3;
