# Plan — Local production runtime (build ≠ start, two databases)

**Status:** Done  
**Last updated:** 2026-08-19  
**Tracking:** Follow-up to Phase 4 / [ADR-0011](../adr/0011-local-production-host.md)  
**Depends on:** Phase 4 Done, frontend local production SPA (`vynno` ADR-0014)

---

## Summary

Make the owner’s-machine host a cheap daily driver, matching the frontend:

- `scripts/build` compiles `bin/vynno-api`. `scripts/start` does not rebuild.
- One Compose Postgres, two databases: `vynno` (daily binary) and `vynno_dev` (`scripts/dev`, seed, reset).
- The API process does not create users. Production starts with schema only. The first account is `POST /v1/auth/register` from the SPA.

No `/v1` contract changes.

## Why now

`scripts/start` rebuilt on every launch. Seed, reset, `go run`, the release binary, and frontend Playwright all shared database `vynno`. `cmd/api` bootstrapped `alexdev` on an empty database, which would put a sample account into daily history.

## Constraints

- No new endpoints, fields, query params, or error codes.
- Do not `docker compose down -v`.
- `internal/domain` still imports neither Gin nor the database driver.
- Seed and reset must refuse database `vynno`.
- Do not add a production wipe command.

## Approach

1. Amend [ADR-0011](../adr/0011-local-production-host.md). Runbook: [../local-production.md](../local-production.md).
2. `scripts/start` requires `bin/vynno-api`, starts Compose, waits, runs the binary (`GIN_MODE=release`). Bind `127.0.0.1:8080`.
3. `scripts/stop` stops the API only. `--postgres` also `docker compose stop` (volume kept).
4. `scripts/dev` runs `go run ./cmd/api` against `DEV_DATABASE_URL` on `127.0.0.1:8081`.
5. `ensure_postgres` creates `vynno_dev` if missing (existing volumes never re-run init scripts).
6. Remove `Bootstrap` from `cmd/api`. `BOOTSTRAP_*` is playground-only (`cmd/devdata`).
7. `cmd/devdata` refuses any database name other than `vynno_dev`.
8. Backup/restore stay on `vynno`.

One-time on this machine: copy current `vynno` (playground) into `vynno_dev`, drop/create empty `vynno`, first `scripts/start` migrates with no rows.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Seed against `vynno` | Wipe daily history | Go guard + script overlay of `DEV_DATABASE_URL` |
| `down -v` | Wipes both databases | Scripts never pass `-v`; runbook |
| Playwright vs `:8080` | `e2e_*` users in production | Dev API on `:8081`; `E2E_API_BASE` |
| API listens on all interfaces | LAN can hit public register | Default `ADDR=127.0.0.1:8080` |

## Out of scope

Cloud host, Dockerfile, TLS, launchd, log rotation, second Compose service, disabling register, frontend code changes.

## Exit checklist

- [x] ADR-0011 amended; this plan; runbook; README / AGENTS.md / handoff / `.env.example`
- [x] `scripts/start` does not build; missing binary dies
- [x] `scripts/stop` leaves Postgres up unless `--postgres`
- [x] `vynno` and `vynno_dev`; seed/reset refuse `vynno`
- [x] `cmd/api` does not bootstrap accounts
- [x] `go test ./...` green
- [x] This plan **Done**

## Related

- [../adr/0011-local-production-host.md](../adr/0011-local-production-host.md)
- [../local-production.md](../local-production.md)
- [dev-data.md](./dev-data.md)
- [phase-4-production.md](./phase-4-production.md)
