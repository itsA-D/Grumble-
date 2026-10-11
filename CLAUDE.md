# Grumble

Public board of crowdsourced frustrations, ranked by urgency, with a paid builder layer. Go (latest stable) + PostgreSQL, one modular-monolith binary run as a single container: chi, templ, HTMX, Tailwind, pgx + sqlc, goose, River.

<!-- Maintainer notes (stripped from Claude's context):
- Keep this file under 200 lines. Add a line only when Claude has made the same mistake twice, a review caught something it should have known, or a new teammate would need it.
- Hard rules that must never be skipped belong in CI, not here. This file is guidance, not enforcement.
-->

## Status

Greenfield. The repository is built in milestones M0 to M4 from `PRD.md`. Requirement IDs (for example FR-VOTE-1) in code comments, commits and PR titles point to `PRD.md`. System design, module map and invariants are in `ARCHITECTURE.md`. Read the relevant section of those files before starting a feature; do not copy them into this file.

## Commands

Use `task` (go-task, pinned as a Go tool) only, and run it with `go tool task` if it is not on `PATH`. All tasks are cross-platform (Windows, Linux, macOS). These tasks must exist in `Taskfile.yml`; if one is missing, add it rather than inventing another command.

- `task generate`: templ generate, sqlc generate, Tailwind build (fetches the pinned CLI)
- `task dev`: run the app at http://localhost:3000 with reload (air) and Tailwind watch
- `task build`: build the static binary (must pass before any PR)
- `task lint`: golangci-lint
- `task vet`: `go vet ./...`
- `task test`: unit tests
- `task test:integration`: repository/job tests on Postgres via testcontainers
- `task test:e2e`: Playwright end-to-end tests
- `task check`: lint, vet, test and build (the local gate)
- `task db:up` / `task db:down`: local Postgres in Docker Compose
- `task db:reset`: recreate the database, run migrations and seed
- `task migrate`: run the migration runner
- `task seed`: load labelled seed content and local fixtures
- `task tools:update`: refresh pinned tool versions in `go.mod`

Before declaring a task done, run `task check`. For changes to routes or forms also run `task test:e2e`; for changes that touch SQL also run `task test:integration`.

## Layout (details in `ARCHITECTURE.md`)

- `cmd/app/`: the subcommands `serve`, `worker`, `migrate`, `seed`
- `internal/transport/web/`: HTTP only; thin handlers, view models, templ views, SEO
- `internal/service/`: use cases; one package per capability; declares the interfaces it needs
- `internal/domain/`, `internal/scoring/`: pure entities, rules and scoring, no I/O
- `internal/store/postgres/`: sqlc output, repositories, TxManager
- `internal/platform/`: adapters (config, db, email, llm, billing, turnstile, ratelimit, cache, clock, idgen, realtime)
- `internal/app/`: composition root, manual dependency injection
- `db/migrations/` (goose SQL), `db/queries/` (sqlc), `web/`, `deploy/`, `docs/adr/`

## Rules that must hold

- Dependency direction is `transport` → `service` → `domain`; `store` and `platform` implement interfaces their consumer declares. `domain` and `scoring` import nothing from the module and do no I/O.
- Clients never write to the database; there is no public database API. Public reads see only `status = 'published'`, through dedicated repository methods and the `published_pains` view. There is no RLS.
- Never store a raw IP address. Use the salted `source_hash` from `internal/service/identity`, rotated monthly.
- Render user text as escaped plain text only (templ escapes it). Never mark user content as safe.
- Prices and plans live only in `internal/domain/billing/plans.go`. Scoring weights live only in `internal/scoring/weights.go` and `scoring_versions`. Never hard-code either elsewhere.
- Counters and `urgency` update in the same transaction as the change, through the pure `scoring` package. No business logic in database triggers.
- Entitlements change only in the verified billing webhook handler.
- Import provider SDKs (model, email, billing, Turnstile) only inside their `internal/platform` adapter.
- Page titles and meta descriptions must never contain a count of zero. Build all SEO metadata through `transport/web/seo`.
- AI briefs use stored signals only and are always labelled as AI-generated. Never generate or paraphrase pains with AI.
- Every moderator action writes `moderation_log` in the same transaction.

## Code conventions

- `go vet` and golangci-lint clean; no `interface{}`/`any` at boundaries — parse into typed structs.
- Validate every request body, query and env value; request and response structs live beside the handler or service and are shared as Go types.
- Named types, one concept per file, files under about 300 lines, exported names documented.
- Handlers parse and validate, call exactly one service method, and map the result or error to a response. No SQL and no business rules in handlers.
- Errors: typed domain errors (`ErrNotFound`, `ErrConflict`, `ErrValidation`, `ErrRateLimited`, `ErrForbidden`) wrapped with `%w`; one place maps them to HTTP status and user-facing messages. Never leak internals or stack traces.
- Every I/O function takes `context.Context` first. Set timeouts on the server, outbound clients and per database call.
- A `TxManager` runs a unit of work in one transaction; services never manage `pgx.Tx` directly.
- Comments explain why, not what. Reference the requirement ID where a rule comes from, for example `// FR-SEO-3`.
- Dates are stored in UTC (`timestamptz`) and formatted at the edge.
- Tailwind with design tokens; no hard-coded hex colours. Support light and dark.
- Accessible by default: semantic elements, labelled inputs, visible focus, a text alternative for the map.

## Database and migrations

- Create migrations with `go tool goose create <name> sql`; never edit an applied migration.
- After a migration: `task db:reset`, then `task test:integration`.
- Counters and `urgency` are updated in the same transaction as the change, through `internal/scoring`. Do not update them from anywhere else, and do not add database triggers that carry business rules.
- Seed data is flagged `seeded = true` and must never be presented as organic.

## Testing expectations

- Unit tests: scoring (reproduce the `PRD.md` section 9 examples), PII detection, moderation rules, slug generation, hub indexability, rate limiter.
- Integration tests (testcontainers): every repository, vote uniqueness, published-only reads (unpublished rows never leak), webhook idempotency and signatures, job handlers.
- HTTP tests with `httptest`: every handler group, including HTMX and non-HTMX paths and error mapping.
- End to end: submit, see it in the feed, vote, hub render with server HTML, flag, sandbox checkout.
- Add or update a test with every behaviour change. Do not delete a failing test to make a build pass; fix the cause or say why it is wrong.

## Git and workflow

- Branches: `feat/<id>-short-name`, `fix/<short-name>`. Commits follow Conventional Commits (`feat(voting): ...`) and cite requirement IDs where relevant.
- Keep changes small and focused. Do not refactor unrelated code or reformat files you did not change.
- Use plan mode and propose the approach before changing: the database schema, the published-only read boundary, scoring weights, billing, moderation logic, or anything in `ARCHITECTURE.md` section 8.
- Changes to the invariants or module boundaries need an ADR in `docs/adr/` (`NNNN-title.md`: context, decision, consequences) and a matching edit to `ARCHITECTURE.md`.
- Never commit secrets, `.env*` files, or generated credentials. Update `.env.example` when adding a variable, and declare it in `internal/platform/config`.

## Environment

Variables are declared and validated in `internal/platform/config`; startup fails if one is missing or invalid. Copy `.env.example` to `.env.local`. Groups: database (`DATABASE_URL`, `DATABASE_DIRECT_URL`, pool size), identity (`ANON_ID_SECRET`, `IP_HASH_SALT`), trusted proxy (`TRUSTED_PROXY_HEADER`, `TRUSTED_PROXIES`), Turnstile, rate limit (in-memory or Redis), model API key and two model names (fast, brief), email, billing, realtime (`REALTIME_MODE`, `SSE_*`, `MAP_REFRESH_MIN_INTERVAL`), metrics, `SITE_URL`, `PRODUCT_NAME`. Check current model names in the provider's documentation before changing defaults.

## Content and product guardrails

- This product is modelled on a public reference site. Use original name, copy and visual design. Do not copy text, logos or layout from it.
- Hub intro text and FAQs are original. Do not auto-generate filler to fill empty hubs; an empty hub is noindex until it has 5 pains.
- Pain text may contain sensitive personal material. Do not log pain bodies or comment bodies; log ids only.
- Self-harm related submissions are held and never published; follow the handling in `PRD.md` FR-MOD-8.

## When unsure

Ask a short question rather than guessing when a requirement is ambiguous, conflicts with `PRD.md`, or would change an invariant. If `PRD.md` and `ARCHITECTURE.md` disagree, stop and report it; do not pick one silently. When you finish, state what changed, which requirement IDs it covers, and what you ran to verify it.
