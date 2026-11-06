-- bookmarks.sql
-- Saved frustrations are private to the anonymous identity. Every read joins
-- published_pains so a removed or unpublished pain disappears from the list.
-- The composite primary key makes saving idempotent.

-- name: InsertBookmark :execrows
-- Returns whether a new row was inserted. The SELECT source enforces that only
-- published pains can be saved; unpublished ids simply insert nothing.
INSERT INTO bookmarks (owner_key, pain_id)
SELECT @owner_key, id FROM published_pains WHERE id = @pain_id
ON CONFLICT (owner_key, pain_id) DO NOTHING;

-- name: DeleteBookmark :execrows
DELETE FROM bookmarks
WHERE owner_key = @owner_key AND pain_id = @pain_id;

-- name: HasBookmark :one
SELECT EXISTS (
    SELECT 1
    FROM bookmarks b
    JOIN published_pains p ON p.id = b.pain_id
    WHERE b.owner_key = @owner_key AND b.pain_id = @pain_id
);

-- name: CountBookmarks :one
SELECT count(*)
FROM bookmarks b
JOIN published_pains p ON p.id = b.pain_id
WHERE b.owner_key = @owner_key;

-- name: ListBookmarkIDs :many
SELECT b.pain_id
FROM bookmarks b
JOIN published_pains p ON p.id = b.pain_id
WHERE b.owner_key = @owner_key
ORDER BY b.created_at DESC, b.pain_id DESC;

-- name: ListBookmarks :many
-- A saved page reads published pains only, newest save first.
SELECT p.id, p.slug, p.body, p.category, p.country, p.status, p.frequency,
       p.disruption, p.would_pay, p.monthly_amount, p.tried, p.vote_count,
       p.urgency, p.seeded, p.created_at, p.published_at
FROM bookmarks b
JOIN published_pains p ON p.id = b.pain_id
WHERE b.owner_key = @owner_key
ORDER BY b.created_at DESC, p.id DESC
LIMIT @page_limit;
