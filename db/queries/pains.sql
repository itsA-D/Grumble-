-- pains.sql
-- Repository reads are the only public lens on pains (ADR 0001): every public
-- query reads through published_pains. Writes and moderation flows touch the
-- pains table directly.

-- name: CreatePain :one
INSERT INTO pains (
    id, slug, body, category, country, status,
    frequency, disruption, would_pay, monthly_amount, tried,
    source_hash, anon_id, seeded, scoring_version
) VALUES (
    $1, $2, $3, $4, $5, 'pending',
    $6, $7, $8, $9, $10,
    $11, $12, $13, $14
)
RETURNING *;

-- name: SetPainStatus :exec
UPDATE pains
SET status = $2,
    published_at = CASE WHEN $2 = 'published' THEN COALESCE(published_at, now()) ELSE published_at END,
    updated_at = now()
WHERE id = $1;

-- name: UpdatePainCounters :exec
UPDATE pains
SET vote_count = $2,
    urgency = $3,
    updated_at = now()
WHERE id = $1;

-- name: GetPainForVote :one
-- Locks the pain row so concurrent votes on the same pain serialise and cannot
-- lose a counter update. Only published pains accept votes (checked by caller).
SELECT id, status, vote_count, frequency, disruption, would_pay, monthly_amount
FROM pains
WHERE id = $1
FOR UPDATE;

-- name: PainSlugExists :one
SELECT EXISTS (SELECT 1 FROM pains WHERE slug = $1);

-- name: GetPublishedBySlug :one
SELECT * FROM published_pains WHERE slug = $1;

-- name: ListPublishedFeedLatest :many
-- Keyset page of published pains, newest first (FR-FEED-1). Empty @category and
-- @search mean "no filter"; search is a literal substring, so user input with
-- % or _ is not treated as a wildcard. The caller passes a far-future
-- @after_created and a nil @after_id to fetch the first page.
SELECT * FROM published_pains
WHERE (@category::text = '' OR category = @category::text)
  AND (@search::text = '' OR position(lower(@search::text) in lower(body)) > 0)
  AND (created_at, id) < (@after_created::timestamptz, @after_id::uuid)
ORDER BY created_at DESC, id DESC
LIMIT @page_limit;

-- name: ListPublishedFeedTopVotes :many
-- Keyset page of published pains, most voted first (FR-FEED-1). The first-page
-- sentinel is @after_votes = max int, @after_created far in the future and a
-- nil @after_id, so every row compares less than the cursor.
SELECT * FROM published_pains
WHERE (@category::text = '' OR category = @category::text)
  AND (@search::text = '' OR position(lower(@search::text) in lower(body)) > 0)
  AND (vote_count, created_at, id) < (@after_votes::int, @after_created::timestamptz, @after_id::uuid)
ORDER BY vote_count DESC, created_at DESC, id DESC
LIMIT @page_limit;

-- name: ListPublishedTrendingCandidates :many
-- Candidates for the trending tab. The pure scoring package finishes the hot
-- rank in Go; this query only supplies the inputs plus the 7-day vote window.
-- comments_7d arrives with the comments table in M2 and is treated as zero.
-- @category and @search narrow the candidate window the same way as the
-- keyset tabs.
SELECT p.id, p.slug, p.body, p.category, p.country, p.status, p.frequency,
       p.disruption, p.would_pay, p.monthly_amount, p.tried, p.vote_count,
       p.urgency, p.seeded, p.created_at, p.published_at,
       (SELECT count(*) FROM votes v
         WHERE v.pain_id = p.id
           AND v.created_at >= now() - interval '7 days') AS votes_7d
FROM published_pains p
WHERE (@category::text = '' OR p.category = @category::text)
  AND (@search::text = '' OR position(lower(@search::text) in lower(p.body)) > 0)
ORDER BY p.created_at DESC
LIMIT @page_limit;

-- name: ListPublishedTrendingWindow :many
-- Candidate shortlist for a trending window. Postgres bounds the work to the
-- 50 most-voted/urgent published pains created inside the window; the pure
-- scoring package then applies the hot rank and its deterministic tie-breaks.
-- votes_window counts votes inside @window_start.
SELECT p.id, p.slug, p.body, p.category, p.country, p.status, p.frequency,
       p.disruption, p.would_pay, p.monthly_amount, p.tried, p.vote_count,
       p.urgency, p.seeded, p.created_at, p.published_at,
       (SELECT count(*) FROM votes v
         WHERE v.pain_id = p.id
           AND v.created_at >= @window_start::timestamptz) AS votes_window
FROM published_pains p
WHERE p.created_at >= @window_start::timestamptz
ORDER BY p.vote_count DESC, p.urgency DESC, p.created_at DESC, p.id DESC
LIMIT @page_limit;

-- name: ListTrendingCategoryCounts :many
-- Category counts inside the same window, for the trending sidebar.
SELECT category, count(*)::int AS pain_count
FROM published_pains
WHERE created_at >= @window_start::timestamptz
GROUP BY category
ORDER BY pain_count DESC, category
LIMIT @page_limit;

-- name: ListTrendingCountryCounts :many
-- Country counts inside the same window, for the trending sidebar.
SELECT country, count(*)::int AS pain_count
FROM published_pains
WHERE created_at >= @window_start::timestamptz
GROUP BY country
ORDER BY pain_count DESC, country
LIMIT @page_limit;

-- name: ListPublishedByCategory :many
SELECT * FROM published_pains
WHERE category = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: ListPublishedByCountry :many
SELECT * FROM published_pains
WHERE country = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: ListRelatedPublished :many
SELECT * FROM published_pains
WHERE category = $1 AND id <> $2
ORDER BY urgency DESC, vote_count DESC, created_at DESC
LIMIT $3;

-- name: CountPublishedTotal :one
SELECT count(*) FROM published_pains;

-- name: CountPublishedByCategory :one
SELECT count(*) FROM published_pains WHERE category = $1;

-- name: CountPublishedByCountry :one
SELECT count(*) FROM published_pains WHERE country = $1;

-- name: CountPublishedCountries :one
-- Distinct countries on the public board, for the home stat strip.
SELECT count(DISTINCT country) FROM published_pains;

-- name: CountPublishedVotes :one
-- Cached vote total across published pains only.
SELECT COALESCE(sum(vote_count), 0)::bigint FROM published_pains;

-- name: ListPublishedByCategoryWithUrgency :many
-- All published pains with urgency for a category (or all if empty).
-- Used by the country-stats job to recompute the read model.
SELECT country, vote_count, urgency
FROM published_pains
WHERE (@category::text = '' OR category = @category::text)
ORDER BY country, urgency DESC, vote_count DESC;

-- name: CreateSeededPain :one
-- Inserts a published, labelled seed pain. ON CONFLICT (slug) DO NOTHING makes
-- seeding idempotent: a rerun inserts nothing and returns no row.
INSERT INTO pains (
    id, slug, body, category, country, status,
    frequency, disruption, would_pay, monthly_amount, tried,
    source_hash, anon_id, seeded, scoring_version,
    vote_count, urgency, published_at
) VALUES (
    $1, $2, $3, $4, $5, 'published',
    $6, $7, $8, $9, $10,
    $11, NULL, TRUE, $12,
    0, 0, now()
)
ON CONFLICT (slug) DO NOTHING
RETURNING id;

-- name: InsertSeedVotes :execrows
-- Adds synthetic votes for a seeded pain in one statement; ON CONFLICT keeps it
-- idempotent. The caller recomputes counters from the real row count, so
-- vote_count always matches the stored votes.
INSERT INTO votes (pain_id, voter_key, created_at)
SELECT $1, k, now()
FROM unnest($2::text[]) AS k
ON CONFLICT (pain_id, voter_key) DO NOTHING;

-- name: RecentSubmissionFromSource :one
-- Moderation: has this source hash submitted (non-seeded) within the window?
SELECT EXISTS (
    SELECT 1 FROM pains
    WHERE source_hash = ANY($1::text[])
      AND created_at >= now() - $2::interval
      AND seeded = false
);