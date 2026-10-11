-- votes.sql
-- Vote rows are private to the app: they exist to compute counts and the
-- "did I vote" answer for the current identity. No public read touches them
-- directly.

-- name: InsertVote :execrows
-- Returns whether a row was actually added (repeat votes are no-ops).
INSERT INTO votes (pain_id, voter_key)
VALUES ($1, $2)
ON CONFLICT (pain_id, voter_key) DO NOTHING;

-- name: DeleteVote :execrows
DELETE FROM votes
WHERE pain_id = $1 AND voter_key = $2;

-- name: HasVoted :one
SELECT EXISTS (
    SELECT 1 FROM votes
    WHERE pain_id = $1 AND voter_key = $2
);

-- name: CountVotesForPain :one
-- Authoritative vote count; the counter on pains is a cached copy (FR-VOTE-2).
SELECT count(*) FROM votes WHERE pain_id = $1;