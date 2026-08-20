# Product Requirements Document — Vynno API

**Status:** Draft  
**Last updated:** 2026-08-20  
**Product name:** Vynno (formerly DevTime)  
**Repository scope:** Backend only (HTTP API, persistence, auth)

The frontend PRD remains the product-facing document for screens and UX. This PRD covers what the **API repository** must do so that product works with real data.

---

## 1. Vision

Vynno helps a person track where their working time goes — by project, task, and activity type — with a dense, technical UI.

The frontend already implements that UI against a mock HTTP contract. This repository exists so those same requests persist, survive reload, and (later) follow the user across devices.

## 2. Problem

Without a backend:

- Timer and project mutations die with the browser tab (mock workspace is SPA-scoped).
- There is no real identity; login accepts any credentials into `sessionStorage`.
- Multi-device use and any future reporting that is not “this tab” are impossible.

Generic time-tracking APIs do not match the contract the SPA already speaks (verb routes, one live session, archive-not-delete, specific error codes). Implementing that contract is the job.

## 3. Goals

### Product goals

1. Persist projects, sessions, and a profile so reload and a second device see the same data.
2. Enforce session and project lifecycle on the server (the client cannot be trusted).
3. Expose the existing `/v1` JSON contract so the SPA can set `PUBLIC_API_BASE` and delete `/mock/v1`.
4. Add real authentication so the stub login can go away (Phase 3).

### Repository goals (this codebase)

1. Start with the same documentation habit as the frontend: PRD, ADRs, roadmap, plans.
2. Keep the wire format in [api-contract.md](./api-contract.md) as the shared language with the SPA.
3. Choose stack and persistence deliberately (ADRs) before scaffolding.

## 4. Non-goals (this repo / near term)

| Non-goal | Rationale |
| --- | --- |
| Any UI | Frontend repository |
| Team / multi-user workspaces | Single-user product first ([ADR-0006](./adr/0006-single-user-tenancy.md)) |
| Invoicing, payroll, client portals | Out of product scope for v1 |
| Calendar / IDE / Git integrations | Not in the frontend mockups |
| Insights / dashboard aggregation endpoints | Client computes these from the session list |
| Theme, locale, command palette | Device-local on the client |
| Inventing resources not in the contract | Amend the contract first |
| OpenAPI-first or a generated client | Optional later; the SPA already has Valibot schemas |

## 5. Personas

Same as the frontend.

**Primary: Solo developer / engineer**  
Tracks focus and project time for self-management. Expects start/pause/stop to be correct after a refresh.

**Secondary: Freelancer / indie hacker**  
Needs durable project breakdowns and weekly totals; no enterprise admin.

## 6. Success metrics (API MVP)

| Metric | Target |
| --- | --- |
| Contract coverage | Every resource in [api-contract.md](./api-contract.md) implemented with the documented status codes |
| Lifecycle correctness | All domain rules in [domain-model.md](./domain-model.md) enforced in tests, not only in handlers |
| Swap readiness | Frontend can set `PUBLIC_API_BASE=https://…/v1` and exercise Timer + Projects without UI changes |
| Durability | Restarting the API process does not lose projects or stopped sessions |
| Auth (Phase 3) | Unauthenticated writes fail; the SPA can attach credentials in one client |

Quantitative product analytics deferred until the API is deployed.

## 7. Information architecture

The API is the information architecture. There is no second resource model.

| Resource | Role |
| --- | --- |
| `GET /me` | Display profile |
| `PATCH /me` | Update display name |
| `PUT` / `DELETE /me/avatar` | Upload or remove photo |
| `/projects` | Work containers; archive / restore / hard delete |
| `/activity-types` | User-owned activity dictionary (display name + token color) |
| `/sessions` | Timed intervals; start + verb actions |

Primary navigation in the SPA (Timer, Dashboard, Logs, Insights, Projects, Settings) is **not** mirrored as endpoints. Dashboard and Insights read the session list and aggregate on the client.

Detailed mapping: [api-contract.md](./api-contract.md), [frontend-handoff.md](./frontend-handoff.md).

## 8. Functional requirements

Priorities: **P0** = live API the SPA can swap to, **P1** = auth + durability for real use, **P2** = contract extensions the frontend already marked P2, **Later** = new product surface.

### 8.1 Profile

| ID | Requirement | Priority |
| --- | --- | --- |
| ME-1 | `GET /me` returns `ProfileDto` | P0 |
| ME-2 | `displayName`, `handle` required; `avatarUrl` JSON `null` when absent | P0 |
| ME-3 | `PATCH /me` updates `displayName` | P1 |
| ME-4 | `PUT /me/avatar` stores jpeg/png/webp ≤ 1 MiB and returns a loadable `avatarUrl` | P1 |
| ME-5 | `DELETE /me/avatar` clears to `null` | P1 |

### 8.2 Projects

| ID | Requirement | Priority |
| --- | --- | --- |
| PRJ-1 | `GET /projects` omits archived unless `includeArchived=true` | P0 |
| PRJ-2 | `GET /projects/:id` returns archived projects (logs need the label) | P0 |
| PRJ-3 | `POST /projects` → `201` `ProjectDto`; name required; `code` optional | P0 |
| PRJ-4 | `PATCH /projects/:id`; `code: null` clears the chip | P0 |
| PRJ-5 | `POST …/archive` and `POST …/restore` | P0 |
| PRJ-6 | `DELETE /projects/:id` → `204` only when zero sessions reference it | P0 |
| PRJ-7 | Cannot archive or hard-delete the last active project | P0 |
| PRJ-8 | `code` unique case-insensitively among non-deleted projects, when set | P0 |
| PRJ-9 | `GET /projects/:id/session-count` → `{ "count": number }` | P0 |
| PRJ-10 | Name 1–80 chars after trim; code `^[A-Z0-9-]{1,8}$` when set | P0 |

### 8.3 Sessions

| ID | Requirement | Priority |
| --- | --- | --- |
| SES-1 | `GET /sessions` newest-first; optional `status` (comma list) and `limit` | P0 |
| SES-2 | `GET /sessions/active` returns the active **or paused** session; idle → `404 session_not_active` | P0 |
| SES-3 | `GET /sessions/:id` | P0 |
| SES-4 | `POST /sessions` → `201`; fails `409 session_already_active` if one is live | P0 |
| SES-5 | Cannot start on a missing (`404`) or archived (`409 project_archived`) project | P0 |
| SES-6 | `POST …/pause`, `…/resume`, `…/stop` — no generic `PATCH status` | P0 |
| SES-7 | Pause accounting: on resume/stop-from-paused, add `now - pausedAt` into `pausedMs` and clear `pausedAt` | P0 |
| SES-8 | Empty/whitespace note becomes `"Untitled session"` | P0 |
| SES-9 | Restart-from-recent is a new `POST /sessions`, not a resume of a stopped row | P0 |
| SES-10 | Edit or delete a stopped session | P2 (LOG-6; needs contract amendment) |
| SES-11 | Manual time entry without running the timer | P2 (LOG-7; needs contract amendment) |
| SES-12 | `activityTypeId` is an optional FK to a user-owned activity type | P1 |

### 8.6 Activity types

| ID | Requirement | Priority |
| --- | --- | --- |
| ACT-1 | `GET /activity-types` returns `{ items }` sorted by name | P1 |
| ACT-2 | `POST /activity-types` → `201`; `name` is a display label; `color` is a token | P1 |
| ACT-3 | `PATCH /activity-types/:id` updates name and/or color | P1 |
| ACT-4 | `DELETE /activity-types/:id` → `204` only when zero sessions reference it | P1 |
| ACT-5 | Register does not seed types; list may be empty | P1 |

### 8.4 Auth

| ID | Requirement | Priority |
| --- | --- | --- |
| AUTH-1 | Decide mechanism ([ADR-0008](./adr/0008-authentication.md)) — HttpOnly cookie + remember-me | P1 |
| AUTH-2 | Unauthenticated access to reads and writes is rejected (`unauthorized`) | P1 |
| AUTH-3 | SPA attaches credentials in `ApiClient` only (`credentials: 'include'`) | P1 |
| AUTH-4 | CORS / cookie / token choices documented in the auth ADR | P1 |

### 8.5 Platform

| ID | Requirement | Priority |
| --- | --- | --- |
| PLAT-1 | Process restart does not lose data | P0 |
| PLAT-2 | Health check for deploy | P1 |
| PLAT-3 | Structured errors only via the contract envelope | P0 |
| PLAT-4 | Observability (request logs, error rates) | P1 |
| PLAT-5 | Backups | P1 |

## 9. Non-functional requirements

| Area | Requirement |
| --- | --- |
| Correctness | Domain rules are server-side and tested; the client is not the system of record |
| Contract | JSON camelCase; lists `{ items }`; absent optionals are JSON `null`; ISO-8601 timestamps |
| IDs | Opaque strings; do not require the frontend’s `proj-` / `sess-` prefixes |
| Time | Store UTC ISO-8601. Display timezone is the client. |
| Security | No secrets in the frontend. Auth design in ADR-0008. |
| Compatibility | A contract change is a paired change with the frontend schemas |
| Performance | v1 assumes the full session list fits in one `GET /sessions` (no cursors yet) |

## 10. Domain summary

See [domain-model.md](./domain-model.md).

Core concepts:

- **Project** — work container with color, optional code, archive flag
- **Time session** — timed interval with `active` / `paused` / `stopped`
- **Activity type** — optional user-owned dictionary row (display name + token color)
- **Profile** — display name, handle, optional avatar

## 11. Assumptions

1. Product name is **Vynno** ([ADR-0007](./adr/0007-product-name.md)).
2. Single concurrent live session (running or paused).
3. Single-user product for v1 ([ADR-0006](./adr/0006-single-user-tenancy.md)).
4. Activity types are a per-user dictionary ([ADR-0012](./adr/0012-activity-types.md)). Display name stored as typed. Empty on register.
5. Insights, prefs, theme, and locale stay on the client until the contract says otherwise.
6. The frontend mock is disposable and is **not** the system of record.

## 12. Open questions

Canonical table: [open-questions.md](./open-questions.md). Highlights:

| # | Question | Default for now |
| --- | --- | --- |
| 1 | Language / framework | **Resolved:** Go + Gin — [ADR-0001](./adr/0001-backend-stack.md) |
| 2 | Database | **Resolved:** PostgreSQL — [ADR-0009](./adr/0009-persistence.md) |
| 3 | Auth mechanism | **Resolved:** HttpOnly cookie — [ADR-0008](./adr/0008-authentication.md) |
| 4 | Restrict `color` to the UI palette? | No — any `#rrggbb` |
| 5 | Pagination | Not until history size requires it |
| Hosting | **Resolved:** owner’s machine — [ADR-0011](./adr/0011-local-production-host.md) |

## 13. Related documents

- [domain-model.md](./domain-model.md)
- [api-contract.md](./api-contract.md)
- [frontend-handoff.md](./frontend-handoff.md)
- [roadmap.md](./roadmap.md)
- [working-agreement.md](./working-agreement.md)
- [adr/](./adr/)
