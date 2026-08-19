# Plan — Local reset and demo dataset

**Status:** Done  
**Last updated:** 2026-08-19  
**Tracking:** Backend-only operator tooling (not a contract phase)  
**Depends on:** [ADR-0009](../adr/0009-persistence.md), [ADR-0011](../adr/0011-local-production-host.md), [domain-model.md](../domain-model.md)

---

## Summary

Two operator commands against database **`vynno_dev`** only (not daily `vynno`):

- **`./scripts/reset`** — wipe playground tables, then restore `alexdev` + one Identity project.
- **`./scripts/seed`** — wipe, then load a production-like playground: three isolated accounts, each with projects and a history of sessions you can log into from the SPA (dev API on `:8081`).

No new `/v1` resources. `cmd/api` does not auto-load this dataset and does not bootstrap accounts. Historical sessions are written through the store (not HTTP), because v1 cannot create a stopped session without running the timer ([backlog LOG-7](../backlog.md)).

## Why now

Production `vynno` starts empty; the first user is SPA register. Frontend e2e `registerAccount` creates `e2e_<timestamp>_…` users — point those at `vynno_dev` (`scripts/dev` / `E2E_API_BASE`) so they do not land in daily history. Filling charts/logs by hand on the playground is tedious; that is what `scripts/seed` is for.

## Constraints

- No new endpoints, fields, query params, or error codes.
- Do not run against database `vynno`. `cmd/devdata` refuses any other name than `vynno_dev`.
- Do not bootstrap accounts from `cmd/api`.
- Do not `docker compose down -v`.
- Do not truncate `goose_db_version`.
- Domain rules stay enforced in the generated rows, not only in HTTP handlers.
- Single-user product: extra accounts are isolated personal logins, not a workspace.
- Do not add `Truncate` to `store.Store`.

## Approach

1. `internal/devdata` builds a catalog from `time.Now()` (so “today” / “this week” stay true) and applies it through `store.Postgres`.
2. `cmd/devdata` subcommands `reset` and `seed` load `.env`, migrate if needed, wipe app tables, apply.
3. `scripts/reset` and `scripts/seed` confirm (or `--yes`), ensure Postgres, point `DATABASE_URL` at `vynno_dev`, then `go run ./cmd/devdata …`. They do not stop the production API.
4. Wipe is:

   ```sql
   TRUNCATE TABLE
     auth_tokens, avatars, sessions, projects, profiles, users
   RESTART IDENTITY CASCADE;
   ```

5. `seed` always wipes first. `alexdev` keeps `DefaultUserID` and bootstrap credentials so the existing SPA login still works.

### Demo accounts (after `scripts/seed`)

| Username | Password | What you see |
| --- | --- | --- |
| `alexdev` | `BOOTSTRAP_PASSWORD` | Power user, 7 projects (one archived), ~10 weeks of logs, one live session |
| `maya` | `SEED_PASSWORD` (default `local-dev-password`) | Contractor, 4 projects (one archived), idle |
| `rio` | same as Maya | New-ish account, two projects, short history |

Neither script restarts an API. Hit `scripts/dev` (or the SPA against `:8081`) after a wipe. Playground cookies are invalid because `auth_tokens` is wiped. Production sessions on `vynno` are untouched.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Wipe daily `vynno` by habit | Lose real history | `cmd/devdata` refuses any database except `vynno_dev` |
| Seed while `scripts/dev` is up | Mid-request 500s / FK errors | Stop the dev process first; production on `:8080` is a different database |
| Too many sessions | SPA `GET /sessions` on boot gets slow | Cap ~400 for the power user |
| Stale “today” data | Insights look empty after a month | Generate from `time.Now()`, not a dump |
| Bootstrap vs seed clash | Second API start creates a fourth user | `alexdev` keeps `DefaultUserID`; `Bootstrap` no-ops when that row has a password |

## Out of scope

- Auto-reset after frontend e2e
- Generated avatars
- Checked-in SQL dumps as the source of truth
- Seeding via HTTP or a `/v1/admin` route
- Pagination / LOG-6/7 / insights API changes
- (Superseded) a separate Docker *service* for demo vs daily use — two *databases* on the same Compose Postgres shipped in [local-prod-runtime.md](./local-prod-runtime.md)

## Exit checklist

- [x] This plan written; `plans/README.md`, README, AGENTS.md, open-question #15 updated
- [x] `scripts/reset` wipes app tables and leaves bootstrap `alexdev` + Identity
- [x] `scripts/seed` wipes and loads three users with the catalog above
- [x] Both require confirm or `--yes`; both target `vynno_dev` only
- [x] Generator tests cover domain invariants
- [x] `go test ./...` green
- [ ] Manual check: login as each user in the SPA (operator)

## Related

- [../domain-model.md](../domain-model.md)
- [../adr/0006-single-user-tenancy.md](../adr/0006-single-user-tenancy.md)
- [../adr/0011-local-production-host.md](../adr/0011-local-production-host.md)
- [phase-4-production.md](./phase-4-production.md)
