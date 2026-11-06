-- 20261027132841_add_bookmarks.sql
-- Anonymous saved frustrations (PRD FR-BKM-1). The owner key is the same stable
-- anonymous identity used for votes; rows are private to the app and reads join
-- the published_pains view so an unpublished pain never leaks.
-- +goose Up

CREATE TABLE bookmarks (
    owner_key  text        NOT NULL,
    pain_id    uuid        NOT NULL REFERENCES pains (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_key, pain_id)
);

-- One owner's saved list, newest first.
CREATE INDEX bookmarks_owner_created_idx ON bookmarks (owner_key, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS bookmarks;
