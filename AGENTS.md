# Vynno API — agent instructions

## Project

Vynno (formerly DevTime) is a focus-time tracker. **This repository is the backend** — HTTP API, persistence, and auth.

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
- Auth follows [ADR-0008](./docs/adr/0008-authentication.md) (Accepted): HttpOnly cookie, remember-me, optional Bearer for curl/tests.
- Enforce the domain rules in [docs/domain-model.md](./docs/domain-model.md) on the server. The SPA already assumes them.
- Single-user product for v1 ([ADR-0006](./docs/adr/0006-single-user-tenancy.md)). No team workspaces.

### Stack conventions

- Language: Go. Module: `github.com/EmilM32/vynno-api`.
- HTTP: Gin. Handlers live in `internal/httpserver`. Domain rules live in `internal/domain` and must not import Gin or the database driver.
- Validation is hand-written. Do not treat Gin `binding` tags as the source of truth for contract or lifecycle errors.
- Persistence (Phase 2): PostgreSQL, goose SQL migrations, sqlc, `pgx` via `database/sql`. Local DB is Docker Compose. App reads `DATABASE_URL`.
- Wire format is [docs/api-contract.md](./docs/api-contract.md). No extra endpoints, fields, or error codes.
- OpenAPI is generated from `Server.route(...)` in `internal/httpserver` ([ADR-0013](./docs/adr/0013-openapi-swagger.md)). Do not add a hand-written spec or swaggo comments. Contract amendments update `route()` metadata in the same change.
- Auth is Accepted ([docs/adr/0008-authentication.md](./docs/adr/0008-authentication.md)). Session cookie `vynno_session`; do not return the token in JSON.
- Outbound mail is Accepted ([docs/adr/0015-outbound-email.md](./docs/adr/0015-outbound-email.md)): SMTP via `internal/mail`. Register confirmation and password reset: [docs/plans/email.md](./docs/plans/email.md). Do not log one-time codes in `smtp` mode.
- IDs are UUID strings. Opaque on the wire; do not require `proj-` / `sess-` prefixes.

### Useful commands

```sh
cp .env.example .env          # ADDR, DATABASE_URL, DEV_*, SPA_ORIGIN, PUBLIC_API_ORIGIN, MAIL_* / SMTP_*
./scripts/build               # bin/vynno-api
./scripts/start               # daily driver; does not rebuild; database vynno
./scripts/start --detach
./scripts/stop                # production API only
./scripts/stop --postgres     # production API + Compose stop (keeps the volume)
./scripts/dev                 # go run on :8081 → vynno_dev
./scripts/stop --dev          # playground API only (leftover :8081 bind)
# Goose is per-database at process start: start migrates vynno, dev migrates vynno_dev.
# Mailpit UI http://127.0.0.1:8025 (Compose). First daily register needs SMTP.
./scripts/backup              # pg_dump vynno into backups/
./scripts/reset               # wipe vynno_dev → alexdev@vynno.local + Identity
./scripts/seed                # wipe vynno_dev → 3 demo users
./scripts/setup               # git config core.hooksPath .githooks (pre-push runs go test ./...)
go test ./...
gofmt -w .
golangci-lint run ./...
sqlc generate                 # after changing internal/store/queries or migrations
```

Health check: `GET /healthz` → `{"status":"ok"}`. Readiness: `GET /readyz` (Postgres ping). SPA contract is under `/v1`. Operator docs: `GET /swagger/` (Swagger UI) and `GET /openapi.json` (generated from `internal/httpserver` route registration). Open Swagger at `PUBLIC_API_ORIGIN`.

Do not `docker compose down -v` on a machine that holds real data.

Migrations run automatically at process start (goose).

### Docs

Product and architecture: `docs/README.md`. ADRs: `docs/adr/`. Plans: `docs/plans/`.
