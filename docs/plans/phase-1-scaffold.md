# Plan — Phase 1 scaffold

**Status:** Done  
**Last updated:** 2026-08-17  
**Tracking:** Roadmap Phase 1  
**Depends on:** [ADR-0001](../adr/0001-backend-stack.md) Accepted, [ADR-0009](../adr/0009-persistence.md) Accepted, [ADR-0008](../adr/0008-authentication.md) Deferred

---

## Summary

Initialize the Go module, Gin, config, lint/test/CI, a health endpoint, and local PostgreSQL via Docker Compose. No `/v1` resources, no domain tables, no auth.

## Why now

Phase 0 is done. Scaffolding without a written stack would re-litigate tools. This plan is the first code so `dev` / `test` / `lint` exist before Phase 2 implements the contract.

## Constraints

- No endpoints from [../api-contract.md](../api-contract.md). Health is an implementation detail (`GET /healthz`).
- No domain tables or goose migrations yet (Phase 2). Compose only runs an empty Postgres so `DATABASE_URL` is real.
- No auth, CORS credentials, or secrets in git.
- `internal/domain` may exist as a placeholder; it must not import Gin or `pgx`.

## Approach

1. `go mod init github.com/EmilM32/vynno-api`.
2. Layout from ADR-0001: `cmd/api`, `internal/httpserver`, `internal/config`. Optional empty `internal/domain` / `internal/store` packages are fine; do not add SQL yet.
3. Gin router with `GET /healthz` → `200` `{"status":"ok"}`. Not part of the SPA contract.
4. Config from env: `ADDR` (default `:8080`), `DATABASE_URL` (required at boot even if unused in Phase 1, so missing config fails loudly).
5. `docker-compose.yml`: PostgreSQL 16, published port, user/password/db for local only (documented defaults, not production secrets).
6. `.env.example` listing `ADDR` and `DATABASE_URL`. No `.env` in git.
7. `gofmt`, `.golangci.yml`, `go test` with at least one HTTP test that hits `/healthz`.
8. GitHub Actions: lint + test. Postgres service in CI so `DATABASE_URL` can be set (even if unused beyond config parse).
9. Fill `AGENTS.md` “Stack conventions” and “Useful commands”.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Implementing `/v1` “while we’re here” | Dual API / contract drift | Non-goals below; handlers stay health-only |
| Skipping Compose | Phase 2 then invents local Postgres | Compose lands now |
| Gin default logger/recovery only | Fine for Phase 1 | Do not add CORS/auth middleware yet |

## Out of scope

- Domain schema, sqlc, goose migrations (Phase 2).
- Auth / CORS (Phase 3).
- Production host (Phase 4).
- OpenAPI.

## Exit checklist

- [x] `go run ./cmd/api` serves `GET /healthz`
- [x] `docker compose up -d` starts Postgres; `DATABASE_URL` documented
- [x] `go test ./...` and `golangci-lint` green
- [x] CI workflow on push/PR
- [x] `AGENTS.md` lists stack conventions and commands
- [x] Roadmap Phase 1 boxes checked; this plan marked **Done**

## Related

- [../roadmap.md](../roadmap.md)
- [../adr/0001-backend-stack.md](../adr/0001-backend-stack.md)
- [../adr/0009-persistence.md](../adr/0009-persistence.md)
