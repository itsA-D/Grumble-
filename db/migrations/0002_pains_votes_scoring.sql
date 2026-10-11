-- 0002_pains_votes_scoring.sql
-- M1 data model (PRD FR-SUB-1..8, FR-VOTE-1/2, FR-HUB-1, FR-DET-1). Rows are
-- created exclusively by the app in transactions; no triggers carry rules.
-- +goose Up

-- ---------------------------------------------------------------------------
-- pains
-- ---------------------------------------------------------------------------
CREATE TABLE pains (
    id            uuid        PRIMARY KEY,
    slug          text        NOT NULL UNIQUE,
    body          text        NOT NULL CHECK (char_length(body) BETWEEN 20 AND 2000), -- FR-SUB-1
    category      text        NOT NULL CHECK (category IN (
        'business', 'health', 'education', 'housing', 'finance', 'travel', 'tech',
        'relationships', 'food', 'government', 'sports', 'entertainment', 'gaming',
        'parenting', 'pets', 'fitness', 'fashion', 'transportation', 'environment',
        'career', 'shopping', 'other'
    )),
    -- ISO 3166-1 alpha-2; membership is validated by the app taxonomy catalog.
    country       text        NOT NULL CHECK (country ~ '^[A-Z]{2}$'),
    status        text        NOT NULL DEFAULT 'pending' CHECK (status IN (
        'pending', 'published', 'held', 'rejected', 'removed'
    )),
    -- FR-SUB-5 submitter signals (nullable: not provided)
    frequency     text        CHECK (frequency IN ('daily', 'weekly', 'sometimes')),
    disruption    text        CHECK (disruption IN ('mild', 'annoying', 'ruins_my_week')),
    would_pay     boolean     NOT NULL DEFAULT FALSE,
    monthly_amount integer    CHECK (monthly_amount IS NULL OR monthly_amount >= 0),
    tried         text        CHECK (tried IS NULL OR char_length(tried) <= 500),
    -- Counters and urgency are maintained by the scoring package in the same
    -- transaction as the change (invariant: no business logic in triggers).
    vote_count    integer     NOT NULL DEFAULT 0 CHECK (vote_count >= 0),
    urgency       integer     NOT NULL DEFAULT 0 CHECK (urgency BETWEEN 0 AND 100),
    scoring_version integer   NOT NULL DEFAULT 1,
    seeded        boolean     NOT NULL DEFAULT FALSE, -- labelled seed content, never presented as organic
    source_hash   text        NOT NULL, -- salted hash of client address, rotated monthly (NFR-PRIV-1)
    anon_id       text,                 -- anonymous identity, best-effort
    created_at    timestamptz NOT NULL DEFAULT now(),
    published_at  timestamptz,
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Feed reads only published rows (ADR 0001).
CREATE INDEX pains_created_idx     ON pains (created_at DESC)     WHERE status = 'published';
CREATE INDEX pains_votes_idx       ON pains (vote_count DESC, created_at DESC) WHERE status = 'published';
CREATE INDEX pains_category_idx    ON pains (category, created_at DESC) WHERE status = 'published';
CREATE INDEX pains_country_idx     ON pains (country, created_at DESC) WHERE status = 'published';
CREATE INDEX pains_source_hash_idx ON pains (source_hash, created_at DESC);

-- ---------------------------------------------------------------------------
-- votes
-- ---------------------------------------------------------------------------
CREATE TABLE votes (
    pain_id    uuid        NOT NULL REFERENCES pains (id) ON DELETE CASCADE,
    voter_key  text        NOT NULL,   -- "u:<user_id>" or "a:<anon_id>"
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (pain_id, voter_key)
);

-- Toggle lookup and "have I voted" checks.
CREATE INDEX votes_voter_idx ON votes (voter_key, created_at DESC);
-- Feeding the 7-day window used by the trending tab.
CREATE INDEX votes_pain_recent_idx ON votes (pain_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- scoring_versions
-- ---------------------------------------------------------------------------
CREATE TABLE scoring_versions (
    version    integer     PRIMARY KEY,
    frequency  numeric(4,3) NOT NULL,
    disruption numeric(4,3) NOT NULL,
    pay        numeric(4,3) NOT NULL,
    crowd      numeric(4,3) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Baseline weights (PRD section 9.2); live in this table only (invariant).
INSERT INTO scoring_versions (version, frequency, disruption, pay, crowd)
VALUES (1, 0.300, 0.250, 0.250, 0.200)
ON CONFLICT (version) DO NOTHING;

-- ---------------------------------------------------------------------------
-- published_pains: the only public lens on pains (ADR 0001). Repository
-- methods for public reads go through this view; there is no RLS.
-- ---------------------------------------------------------------------------
CREATE VIEW published_pains AS
SELECT id,
       slug,
       body,
       category,
       country,
       status,
       frequency,
       disruption,
       would_pay,
       monthly_amount,
       tried,
       vote_count,
       urgency,
       scoring_version,
       seeded,
       created_at,
       published_at
FROM pains
WHERE status = 'published';

-- +goose Down
DROP VIEW IF EXISTS published_pains;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS pains;
DROP TABLE IF EXISTS scoring_versions;