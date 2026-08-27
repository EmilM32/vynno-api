# Vynno API — agent instructions

## Project

Vynno (formerly DevTime) is a focus-time tracker. **This repository is the backend** — HTTP API, persistence, and auth.

The SvelteKit frontend is a **separate repository** ([`vynno`](https://github.com/EmilM32/vynno)). It already speaks the contract in [docs/api-contract.md](./docs/api-contract.md). Do not invent a parallel API.

| In this repo | Separate (frontend) |
| --- | --- |
| `/v1` JSON API, database, auth | SvelteKit UI, routing, design system |
| Server-side session and project rules | Display, client aggregates, theme, locale |

### Working rules

- Wire change → [docs/api-contract.md](./docs/api-contract.md) first, plus the frontend schemas. Do **not** add endpoints, fields, query params, or error codes that are not in the contract.
- Lifecycle / invariant change → [docs/domain-model.md](./docs/domain-model.md), then the matching ADR if it is a decision with alternatives.
- New expensive choice (stack, host, auth mechanism) → new or amended ADR under `docs/adr/`. New multi-day feature → a plan under `docs/plans/` while the work is in flight; delete the plan once its facts are in the contract, domain, ADR, or runbook.
- Auth follows [ADR-0008](./docs/adr/0008-authentication.md): HttpOnly cookie `vynno_session`, remember-me, optional Bearer for curl/tests. Do not return the token in JSON.
- Enforce domain rules on the server. Single-user v1 ([ADR-0006](./docs/adr/0006-single-user-tenancy.md)). No team workspaces.
- SPA attach: `PUBLIC_API_BASE=/v1`, `credentials: 'include'`, never `Authorization`. List SPA origins in `SPA_ORIGIN`. Frontend e2e must use `E2E_API_BASE=http://localhost:8081/v1` while the daily binary is on `:27182`.
- Do not add `Truncate` to `store.Store`. `cmd/devdata` (reset/seed) refuses any database except `vynno_dev`. Do not add a production wipe script. Do not `docker compose down -v`.

### Stack conventions

- Language: Go. Module: `github.com/EmilM32/vynno-api`.
- HTTP: Gin. Handlers live in `internal/httpserver`. Domain rules live in `internal/domain` and must not import Gin or the database driver.
- Validation is hand-written. Do not treat Gin `binding` tags as the source of truth for contract or lifecycle errors.
- Persistence: PostgreSQL, goose SQL migrations, sqlc, `pgx` via `database/sql`. Local DB is Docker Compose. App reads `DATABASE_URL`.
- Wire format is [docs/api-contract.md](./docs/api-contract.md).
- OpenAPI is generated from `Server.route(...)` in `internal/httpserver` ([ADR-0013](./docs/adr/0013-openapi-swagger.md)). Do not add a hand-written spec or swaggo comments. Contract amendments update `route()` metadata in the same change.
- Outbound mail is [ADR-0015](./docs/adr/0015-outbound-email.md): SMTP via `internal/mail`. Do not log one-time codes in `smtp` mode.
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

- Wire: `docs/api-contract.md`
- Rules: `docs/domain-model.md`
- Decisions: `docs/adr/`
- Runbook: `docs/local-production.md`
- Process: `docs/working-agreement.md`
- Later work: `docs/backlog.md`
