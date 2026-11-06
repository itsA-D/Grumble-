# Grumble

A public, crowdsourced board of real frustrations ranked by urgency and willingness to pay. Built as a modular monolith in Go.

## Overview

Grumble lets anyone post a frustration anonymously, vote on others, and see pain clusters by category and location. Builders can discover high-urgency problems with signals about frequency, disruption, and willingness to pay.

**Key Features:**
- **Anonymous posting** - No account required to submit frustrations
- **Community voting** - Vote on pains to influence urgency ranking
- **Smart scoring** - Urgency score combines submitter signals (frequency, disruption, willingness to pay) with crowd votes
- **Discovery** - Browse by trending/latest/most voted, filter by category and country, explore the world map
- **SEO-friendly** - Server-rendered pages for search engine visibility
- **Pro features** (planned) - Money feed, AI briefs, exports, alerts

See [PRD.md](PRD.md) for full product requirements and [ARCHITECTURE.md](ARCHITECTURE.md) for technical details.

## Tech Stack

| Layer | Technology |
| --- | --- |
| Language | Go 1.26+ |
| HTTP Router | [go-chi/chi/v5](https://github.com/go-chi/chi) |
| Templates | [templ](https://github.com/a-h/templ) (type-safe HTML) |
| Interactivity | [HTMX](https://htmx.org/) + vanilla JS |
| Styling | [Tailwind CSS](https://tailwindcss.com/) (standalone CLI) |
| Database | PostgreSQL 16+ |
| DB Access | [pgx/v5](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev/) |
| Migrations | [goose](https://github.com/pressly/goose) |
| Jobs/Queue | [River](https://riverqueue.com/) (Postgres-backed) |
| Live Updates | SSE with LISTEN/NOTIFY |
| E2E Tests | [Playwright](https://playwright.dev/) |

## Prerequisites

- [Go](https://go.dev/dl/) 1.26 or later
- [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/) (for local Postgres)
- [Task](https://taskfile.dev/) (optional, but recommended) - can be run via `go tool task`

All other tools (templ, sqlc, air, golangci-lint, etc.) are pinned in `go.mod` and run via `go tool`.

## Quick Start

### 1. Clone and setup environment

```bash
git clone <repo-url>
cd Grumble

# Copy environment template
cp .env.example .env.local
```

Edit `.env.local` with your values. For local development, the defaults work if you use the Docker Compose Postgres setup.

### 2. Start database

```bash
# Using Task
task db:up

# Or directly with Docker Compose
docker compose up -d --wait db
```

Postgres will be available on `localhost:5433` (to avoid conflicts with local Postgres on 5432).

### 3. Run migrations and seed data

```bash
# Using Task
task migrate
task seed

# Or directly
go run ./cmd/app migrate
go run ./cmd/app seed
```

### 4. Generate code and build assets

```bash
# Generate templ, sqlc, and CSS
task generate

# Or run individually
go tool templ generate
go tool sqlc generate
go run ./scripts/tw -i web/src/app.css -o web/static/app.css --minify
```

### 5. Run the application

```bash
# Development with hot reload (recommended)
# Opens the app in your browser once the server is ready.
task dev

# Or simple run
task run

# Or build and run binary
task build
./bin/app serve
```

The app will be available at [http://localhost:3000](http://localhost:3000).

## Development

### Common Tasks

| Command | Description |
| --- | --- |
| `task` | List all available tasks |
| `task generate` | Regenerate templ views, sqlc code, and Tailwind CSS |
| `task css:watch` | Watch and rebuild CSS on changes (run alongside `task dev`) |
| `task dev` | Run with live reload via Air, then open the app in the browser |
| `task run` | Run the app directly |
| `task build` | Build production binary to `bin/app` |
| `task test` | Run unit tests |
| `task test:integration` | Run integration tests (requires Docker) |
| `task test:e2e` | Run Playwright E2E tests |
| `task lint` | Run golangci-lint |
| `task vet` | Run go vet |
| `task check:imports` | Check module import boundaries |
| `task check` | Run full local gate (lint, vet, test, imports, build) |

### Project Structure

```
Grumble/
├── cmd/
│   └── app/                # Application entrypoint and subcommands (serve, worker, migrate, seed)
├── db/
│   ├── migrations/         # Database migrations (goose, embedded)
│   └── queries/            # SQL queries for sqlc code generation
├── deploy/
│   ├── Dockerfile          # Production container image
│   └── PRODUCTION_DEPLOY.md
├── docs/
│   └── adr/                # Architecture Decision Records
├── e2e/                    # Playwright end-to-end tests
├── internal/
│   ├── app/                # Composition root, dependency injection
│   ├── domain/             # Entities, value objects, enums, pure domain rules
│   │   ├── moderation/
│   │   ├── pain/
│   │   ├── pii/
│   │   └── taxonomy/
│   ├── platform/           # External adapters (config, logging, db, ratelimit, turnstile, etc.)
│   │   ├── clock/
│   │   ├── config/
│   │   ├── db/
│   │   ├── idgen/
│   │   ├── logging/
│   │   ├── ratelimit/
│   │   └── turnstile/
│   ├── scoring/            # Urgency and trending score calculations (pure functions)
│   ├── service/            # Use cases/business logic per capability
│   │   ├── feed/
│   │   ├── hubs/
│   │   ├── identity/
│   │   ├── pains/
│   │   ├── seed/
│   │   └── voting/
│   ├── store/
│   │   └── postgres/       # Repositories and sqlc-generated code
│   └── transport/
│       └── web/            # HTTP handlers, middleware, routes, SEO, templates
│           ├── seo/
│           └── views/
├── scripts/
│   ├── checkimports/       # Import boundary enforcement
│   └── tw/                 # Tailwind CSS build helper
├── web/
│   ├── src/                # Tailwind CSS source
│   │   └── app.css
│   ├── static/             # Compiled CSS, JS, and static assets
│   └── embed.go            # Static asset embedding
├── .env.example            # Environment variables template
├── go.mod                  # Go module definition
├── Taskfile.yml            # Task runner configuration
├── sqlc.yaml               # sqlc configuration
└── playwright.config.ts    # Playwright configuration
```

### Working with Templates

Templates use [templ](https://templ.guide/). After editing `.templ` files, regenerate:

```bash
go tool templ generate
```

### Working with Database

1. Add a migration in `db/migrations/` (follow existing naming convention)
2. Add queries in `db/queries/`
3. Generate code: `go tool sqlc generate` or `task sqlc`
4. Run migrations: `go run ./cmd/app migrate` or `task migrate`

### CSS Development

```bash
# One-time build
task css

# Watch mode (in separate terminal)
task css:watch
```

## Testing

```bash
# Unit tests
go test ./...

# With race detector
go test -race ./...

# Integration tests (requires Docker)
task test:integration

# E2E tests (requires running app + db)
task test:e2e
```

## Environment Variables

See [.env.example](.env.example) for all available configuration options. Key variables:

| Variable | Description | Default |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://grumble:grumble@localhost:5433/grumble?sslmode=disable` |
| `HTTP_ADDR` | Server listen address | `:3000` |
| `SITE_URL` | Public site URL | `http://localhost:3000` |
| `ANON_ID_SECRET` | Secret for anonymous ID generation | Required (dev defaults in Taskfile) |
| `IP_HASH_SALT` | Salt for IP hashing | Required (dev defaults in Taskfile) |
| `APP_ENV` | Environment (dev/staging/prod) | `dev` |

**Important:** Never commit real secrets. Use `.env.local` for local overrides.

## Deployment

See [deploy/PRODUCTION_DEPLOY.md](deploy/PRODUCTION_DEPLOY.md) for production deployment instructions. The app builds as a static binary (`CGO_ENABLED=0`) in a minimal container.

## Contributing

1. Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand design principles and invariants
2. Follow existing patterns and module boundaries
3. Run `task check` before committing (lint, vet, test, imports, build)
4. Write tests for new functionality
5. Keep changes minimal and focused

## Additional Documentation

- [PRD.md](PRD.md) - Product requirements and specifications
- [ARCHITECTURE.md](ARCHITECTURE.md) - Technical architecture and design decisions
- [CLAUDE.md](CLAUDE.md) - Agent instructions and project conventions
- [docs/](docs/) - Additional documentation including ADRs
- [deploy/](deploy/) - Deployment guides and Dockerfiles
