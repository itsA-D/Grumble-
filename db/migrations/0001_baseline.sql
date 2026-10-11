-- +goose Up
-- Baseline: extensions the schema relies on. Domain tables are added in the
-- milestone that first uses them (see ARCHITECTURE.md section 5); this
-- migration establishes only the shared groundwork so migrations have a
-- starting point and every environment agrees on available extensions.
--
-- pg_trgm powers fuzzy duplicate detection (PRD FR-SUB-7).
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- +goose Down
DROP EXTENSION IF EXISTS pg_trgm;
