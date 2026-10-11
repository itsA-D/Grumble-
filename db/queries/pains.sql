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

-- name: ListPublishedLatest :many
SELECT * FROM published_pains
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: ListPublishedTopVotes :many
SELECT * FROM published_pains
ORDER BY vote_count DESC, created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: ListPublishedTrendingCandidates :many
-- Candidates for the trending tab. The pure scoring package finishes the hot
-- rank in Go; this query only supplies the inputs plus the 7-day vote window.
-- comments_7d arrives with the comments table in M2 and is treated as zero.
SELECT p.id, p.slug, p.body, p.category, p.country, p.status, p.frequency,
       p.disruption, p.would_pay, p.monthly_amount, p.tried, p.vote_count,
       p.urgency, p.created_at, p.published_at,
       (SELECT count(*) FROM votes v
         WHERE v.pain_id = p.id
           AND v.created_at >= now() - interval '7 days') AS votes_7d
FROM published_pains p
ORDER BY p.created_at DESC
LIMIT $1;

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

-- name: RecentSubmissionFromSource :one
-- Moderation: has this source hash submitted (non-seeded) within the window?
SELECT EXISTS (
    SELECT 1 FROM pains
    WHERE source_hash = ANY($1::text[])
      AND created_at >= now() - $2::interval
      AND seeded = false
);