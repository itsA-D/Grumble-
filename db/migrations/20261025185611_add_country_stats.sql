-- 0003_add_country_stats.sql
-- M2 map read model (PRD FR-MAP-1). Recomputed periodically from published
-- pains only; idempotent so running twice changes nothing.
--
-- `category` is NOT NULL with '' meaning "all categories". A nullable column
-- would defeat the (country, category) primary key, because Postgres treats
-- NULLs as distinct and ON CONFLICT would never match the aggregate row.
-- +goose Up

CREATE TABLE country_stats (
    country       text        NOT NULL CHECK (country ~ '^[A-Z]{2}$'),
    category      text        NOT NULL DEFAULT '',   -- '' means "all categories"
    pain_count    integer     NOT NULL DEFAULT 0 CHECK (pain_count >= 0),
    vote_count    integer     NOT NULL DEFAULT 0 CHECK (vote_count >= 0),
    avg_urgency   integer     NOT NULL DEFAULT 0 CHECK (avg_urgency BETWEEN 0 AND 100),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (country, category)
);

CREATE INDEX country_stats_category_idx ON country_stats (category);
CREATE INDEX country_stats_pain_count_idx ON country_stats (pain_count DESC) WHERE category = '';

-- Helper view: one row per country across all categories (the '' aggregate row).
CREATE VIEW country_stats_totals AS
SELECT country, pain_count, vote_count, avg_urgency
FROM country_stats
WHERE category = '';

-- +goose Down
DROP VIEW IF EXISTS country_stats_totals;
DROP TABLE IF EXISTS country_stats;
