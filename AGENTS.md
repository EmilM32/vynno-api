# Vynno API — agent instructions

## Project

Vynno (formerly DevTime) is a focus-time tracker. **This repository is the backend** — HTTP API, persistence, and (later) auth.

The SvelteKit frontend is a **separate repository** ([`vynno`](https://github.com/EmilM32/vynno)). It already speaks the contract in [docs/api-contract.md](./docs/api-contract.md). Do not invent a parallel API.

| In this repo | Separate (frontend) |
| --- | --- |
| `/v1` JSON API, database, auth | SvelteKit UI, routing, design system |
| Server-side session and project rules | Display, client aggregates, theme, locale |

### Working rules

- Read [docs/README.md](./docs/README.md) and [docs/working-agreement.md](./docs/working-agreement.md) before changing product or architecture.
- Docs-first: update PRD / ADR / plan / contract **before** or **with** the code they describe. Large work gets a plan under `docs/plans/`.
- Do **not** add endpoints, fields, query params, or error codes that are not in [docs/api-contract.md](./docs/api-contract.md). Amend the contract first (and the frontend schemas once both repos exist).
- Do **not** pick a language, framework, database, or host unless [ADR-0001](./docs/adr/0001-backend-stack.md) / [ADR-0009](./docs/adr/0009-persistence.md) are Accepted (or the user explicitly decides them).
- Do **not** implement auth until [ADR-0008](./docs/adr/0008-authentication.md) is Accepted.
- Enforce the domain rules in [docs/domain-model.md](./docs/domain-model.md) on the server. The SPA already assumes them.
- Single-user product for v1 ([ADR-0006](./docs/adr/0006-single-user-tenancy.md)). No team workspaces.

### Stack conventions

- Language: Go. Module: `github.com/EmilM32/vynno-api`.
- HTTP: Gin. Handlers live in `internal/httpserver`. Domain rules live in `internal/domain` and must not import Gin or the database driver.
- Validation is hand-written. Do not treat Gin `binding` tags as the source of truth for contract or lifecycle errors.
- Persistence (Phase 2): PostgreSQL, goose SQL migrations, sqlc, `pgx` via `database/sql`. Local DB is Docker Compose. App reads `DATABASE_URL`.
- Wire format is [docs/api-contract.md](./docs/api-contract.md). No extra endpoints, fields, or error codes.
- Auth is Deferred ([docs/adr/0008-authentication.md](./docs/adr/0008-authentication.md)). Do not add login routes or credential middleware until that ADR is Accepted.
- IDs are UUID strings. Opaque on the wire; do not require `proj-` / `sess-` prefixes.

### Useful commands

```sh
docker compose up -d          # local PostgreSQL 16
cp .env.example .env          # ADDR + DATABASE_URL
go run ./cmd/api              # migrate + seed + listen on :8080
go test ./...
gofmt -w .
golangci-lint run ./...
sqlc generate                 # after changing internal/store/queries or migrations
```

Health check: `GET /healthz` → `{"status":"ok"}`. SPA contract is under `/v1`.

Migrations run automatically at process start (goose).

### Docs

Product and architecture: `docs/README.md`. ADRs: `docs/adr/`. Plans: `docs/plans/`.
