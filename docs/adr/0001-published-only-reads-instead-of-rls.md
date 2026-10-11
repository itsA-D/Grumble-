# 0001. Public reads via dedicated methods, not Row Level Security

- Status: Accepted
- Date: 2026-10-10
- Amends: ARCHITECTURE.md section on the published-only read boundary; supersedes the RLS approach in the original Next.js/Supabase plan.

## Context

The original plan (a serverless TypeScript stack) exposed Postgres directly to
clients and used Row Level Security (RLS) to constrain what anonymous clients
could read. PainBoard is now a Go modular monolith: the only writer and reader
of the database is the server process, and there is no public database API.
RLS adds a second, harder-to-test authorisation layer that duplicates what the
Go repository layer already enforces, and it is easy to get subtly wrong
(permissive policies, missing `FORCE ROW LEVEL SECURITY`, session-variable
leaks).

## Decision

- There is **no RLS**. The database is reachable only from trusted server code.
- Public reads are filtered to `status = 'published'` **in dedicated repository
  methods** and, where a denormalised read model helps, through the
  `published_pains` view. Unpublished rows are never returned to public code
  paths.
- The published-only boundary is protected by integration tests that assert
  unpublished rows never leak through any public read.

## Consequences

- One authorisation boundary lives in typed Go code that can be unit- and
  integration-tested; no reliance on database session state or policy parsing.
- Repository methods must be explicit about their visibility: "public" and
  "internal" queries are separate functions, not a flag that is easy to forget.
- Any future public database API would reintroduce the need for RLS; that would
  require a new ADR.
- `db/queries/*.sql` and the `published_pains` view carry a comment pointing at
  this decision and PRD NFR-SEC-2.
