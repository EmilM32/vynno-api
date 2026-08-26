# Roadmap — Vynno API

**Status:** Draft  
**Last updated:** 2026-08-26  
**Scope:** This repository only (API). Frontend is a separate project.

---

## Phase overview

| Phase | Name | Deliverable | Code? |
| --- | --- | --- | --- |
| **0** | Planning | PRD, domain, contract snapshot, ADRs, working agreement | Docs only |
| **1** | Scaffold | Runtime + lint/test + health, after stack ADR | Yes |
| **2** | Contract v1 | Persist + implement `/me`, projects, sessions | Yes |
| **3** | Auth | Real identity; SPA can leave the login stub | Yes |
| **4** | Production | Observability, backups, secrets, CORS | Yes |
| **5** | Later | Prefs, insights — contract first. Pagination shipped. | Later |

---

## Phase 0 — Planning

**Done when:**

- [x] `docs/` published (PRD, domain, contract, roadmap, ADRs, working agreement)
- [x] Root README and `AGENTS.md` point at docs
- [x] Inherited decisions recorded (repo split, contract, project/session rules, single-user, name)
- [x] [ADR-0001](./adr/0001-backend-stack.md) Accepted (Go + Gin)
- [x] [ADR-0009](./adr/0009-persistence.md) Accepted (PostgreSQL + goose)
- [x] [ADR-0008](./adr/0008-authentication.md) Deferred to Phase 3 with the default “no auth on the wire until then”

**Exit criteria:** Product and architecture clear enough to scaffold without re-litigating the contract. Stack and persistence are chosen. Auth has a direction (implement in Phase 3 is a valid direction).

Plan: [plans/phase-0-planning.md](./plans/phase-0-planning.md) (Done).

---

## Phase 1 — Scaffold

**Goals**

- [x] Initialize the runtime chosen in ADR-0001.
- [x] Lint, format, test runner, CI skeleton.
- [x] Config / env loading (no secrets in git).
- [x] Health endpoint (path is an implementation detail; not part of the SPA contract).
- [x] Fill `AGENTS.md` “Stack conventions” and “Useful commands”.

**Non-goals:** Domain tables, auth, implementing `/v1` resources.

**Exit criteria:** `dev` / `test` / `lint` documented and green on an empty app.

Plan: [plans/phase-1-scaffold.md](./plans/phase-1-scaffold.md).

---

## Phase 2 — Contract v1

**Goals**

- [x] Persistence from ADR-0009; migrations.
- [x] Domain module + tests for [domain-model.md](./domain-model.md) (session and project rules).
- [x] HTTP handlers for every resource in [api-contract.md](./api-contract.md).
- [x] Error envelope and documented codes only.
- [x] Seed or bootstrap at least one profile and one active project so the SPA can start a session.

**Non-goals:** Auth, pagination, insights endpoints, log edit/delete.

**Exit criteria:** Frontend can set `PUBLIC_API_BASE` at this origin `/v1` and complete Timer start/pause/resume/stop and project CRUD without UI changes. Process restart keeps the data.

Plan: [plans/phase-2-contract.md](./plans/phase-2-contract.md) (Done).

---

## Phase 3 — Auth

**Depends on:** ADR-0008 Accepted; contract amendment if new routes or codes are required.

**Goals**

- [x] Implement the chosen mechanism.
- [x] Reject unauthenticated writes (and reads, if that is the decision).
- [x] Document CORS / cookie / token setup for the SPA.
- [x] Frontend Phase 5c: attach credentials on `ApiClient`, delete `/mock/v1`.

**Exit criteria:** Stub login is gone. Two browsers with different credentials do not share data (even if v1 only provisions one real user). Remember-me keeps the session across a browser restart.

Plan: [plans/phase-3-auth.md](./plans/phase-3-auth.md) (Done).

---

## Phase 4 — Production

**Depends on:** [ADR-0011](./adr/0011-local-production-host.md) Accepted.

**Goals**

- [x] Request logging and error reporting.
- [x] Backups (and a restore drill).
- [x] Secrets via env (gitignored `.env`; no cloud secret manager until there is a cloud host).
- [x] Deploy target documented (owner’s machine: binary + Compose Postgres).
- [x] CORS locked to the SPA origin(s) (done in Phase 3; Phase 4 only confirms loopback + `COOKIE_SECURE=false`).

**Exit criteria:** A documented local-prod path; data recoverable with `pg_dump`; the SPA on this machine talks to the running binary.

Plan: [plans/phase-4-production.md](./plans/phase-4-production.md) (Done).

---

## Shipped after Phase 3

### Profile edit (ME-3 / ME-4 / ME-5)

Plan: [plans/profile-avatar.md](./plans/profile-avatar.md) (Done).

- [x] Contract amendment: `PATCH /me`, `PUT` / `DELETE /me/avatar`, public `GET /avatars/:id`
- [x] ADR-0010 avatar storage (BYTEA + public URL)
- [x] API + tests
- [x] Frontend Settings pairing

### Local reset and demo dataset

Plan: [plans/dev-data.md](./plans/dev-data.md).

- [x] `scripts/reset` — wipe `vynno_dev`, restore bootstrap `alexdev@vynno.local` + Identity
- [x] `scripts/seed` — wipe `vynno_dev`, load three isolated accounts with production-like history
- [x] Generator lives in `internal/devdata`; not imported by `cmd/api`

### User-defined activity types

Plan: [plans/activity-types.md](./plans/activity-types.md). ADR: [ADR-0012](./adr/0012-activity-types.md).

- [x] Contract amendment: `/activity-types` CRUD; sessions use `activityTypeId`
- [x] Per-user dictionary; display `name`; token `color`; empty on register
- [x] Frontend pairing (Timer picker, Settings, chips, Insights)

### Session edit, delete, and manual entry

Plan: [plans/session-edit.md](./plans/session-edit.md). Amends [ADR-0005](./adr/0005-session-lifecycle.md).

- [x] Contract amendment: `PATCH` / `DELETE /sessions/:id`, `POST /sessions/manual`
- [x] Any session (including live) may be patched or deleted
- [x] Manual create is always `stopped`; allowed while a timer is running
- [x] Frontend pairing (Logs edit/delete, add entry, Timer live fields)

### Operator Swagger UI

Plan: [plans/swagger.md](./plans/swagger.md). ADR: [ADR-0013](./adr/0013-openapi-swagger.md).

- [x] OpenAPI 3.0 generated from Gin `route()` registration (not a hand-written YAML)
- [x] `GET /swagger/` and `GET /openapi.json` (outside `/v1`)
- [x] CSRF allows `PUBLIC_API_ORIGIN` so same-origin Try-it-out works

### Session list pagination

Plan: [plans/session-pagination.md](./plans/session-pagination.md). ADR: [ADR-0014](./adr/0014-session-list-pagination.md).

- [x] Contract amendment: `GET /sessions` `{ items, nextCursor }`; `limit` default 20, max 100; opaque `cursor`
- [x] Keyset on `(started_at, id)`; Memory store matches
- [x] Frontend pairing (first-page seed, Logs infinite scroll, period drain)

### Email login identifier

Plan: [plans/email-login.md](./plans/email-login.md). Amends [ADR-0008](./adr/0008-authentication.md).

- [x] Contract: `email` on register/login and `ProfileDto`; drop `username` / `handle`; `email_in_use`
- [x] Display name may be empty; chrome shows email when unset
- [x] Frontend pairing (login/register, Settings, SideNav)

### Local production runtime

Plan: [plans/local-prod-runtime.md](./plans/local-prod-runtime.md).

- [x] `scripts/build` vs `scripts/start` (start does not rebuild)
- [x] Databases `vynno` (daily) and `vynno_dev` (seed / reset / `scripts/dev`)
- [x] `cmd/api` does not bootstrap accounts; first user is SPA register

---

## Phase 5 — Later

Only via contract amendments. Candidates: [backlog.md](./backlog.md).

- Prefs persistence
- Insights aggregation endpoints
- Multi-user workspaces

---

## Tracking

Use this roadmap as the checklist. Split Phase 1+ into issues later if useful; no issue tracker is required for Phase 0.

---

## Related documents

- [prd.md](./prd.md)
- [backlog.md](./backlog.md)
- [plans/phase-0-planning.md](./plans/phase-0-planning.md)
- [plans/phase-4-production.md](./plans/phase-4-production.md)
- [adr/0001-backend-stack.md](./adr/0001-backend-stack.md)
- [adr/0011-local-production-host.md](./adr/0011-local-production-host.md)
