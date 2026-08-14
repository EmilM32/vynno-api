# Roadmap — Vynno API

**Status:** Draft  
**Last updated:** 2026-08-14  
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
| **5** | Later | Pagination, prefs, log edit, insights — contract first | Later |

---

## Phase 0 — Planning (current)

**Done when:**

- [x] `docs/` published (PRD, domain, contract, roadmap, ADRs, working agreement)
- [x] Root README and `AGENTS.md` point at docs
- [x] Inherited decisions recorded (repo split, contract, project/session rules, single-user, name)
- [ ] [ADR-0001](./adr/0001-backend-stack.md) Accepted (or explicitly deferred with a default — not applicable; stack must be chosen)
- [ ] [ADR-0009](./adr/0009-persistence.md) Accepted
- [ ] [ADR-0008](./adr/0008-authentication.md) Accepted **or** explicitly deferred to Phase 3 with the default “no auth on the wire until then”

**Exit criteria:** Product and architecture clear enough to scaffold without re-litigating the contract. Stack and persistence are chosen. Auth has a direction (implement in Phase 3 is a valid direction).

Plan: [plans/phase-0-planning.md](./plans/phase-0-planning.md).

---

## Phase 1 — Scaffold

**Goals**

- [ ] Initialize the runtime chosen in ADR-0001.
- [ ] Lint, format, test runner, CI skeleton.
- [ ] Config / env loading (no secrets in git).
- [ ] Health endpoint (path is an implementation detail; not part of the SPA contract).
- [ ] Fill `AGENTS.md` “Stack conventions” and “Useful commands”.

**Non-goals:** Domain tables, auth, implementing `/v1` resources.

**Exit criteria:** `dev` / `test` / `lint` documented and green on an empty app.

---

## Phase 2 — Contract v1

**Goals**

- [ ] Persistence from ADR-0009; migrations.
- [ ] Domain module + tests for [domain-model.md](./domain-model.md) (session and project rules).
- [ ] HTTP handlers for every resource in [api-contract.md](./api-contract.md).
- [ ] Error envelope and documented codes only.
- [ ] Seed or bootstrap at least one profile and one active project so the SPA can start a session.

**Non-goals:** Auth, pagination, insights endpoints, log edit/delete.

**Exit criteria:** Frontend can set `PUBLIC_API_BASE` at this origin `/v1` and complete Timer start/pause/resume/stop and project CRUD without UI changes. Process restart keeps the data.

---

## Phase 3 — Auth

**Depends on:** ADR-0008 Accepted; contract amendment if new routes or codes are required.

**Goals**

- [ ] Implement the chosen mechanism.
- [ ] Reject unauthenticated writes (and reads, if that is the decision).
- [ ] Document CORS / cookie / token setup for the SPA.
- [ ] Frontend Phase 5c: attach credentials on `ApiClient`, delete `/mock/v1`.

**Exit criteria:** Stub login is gone. Two browsers with different credentials do not share data (even if v1 only provisions one real user).

---

## Phase 4 — Production

**Goals**

- [ ] Request logging and error reporting.
- [ ] Backups (and a restore drill).
- [ ] Secrets via env / secret manager, not files in git.
- [ ] Deploy target documented.
- [ ] CORS locked to the SPA origin(s).

**Exit criteria:** A documented deploy path; data recoverable; the SPA talks to the deployed origin.

---

## Phase 5 — Later

Only via contract amendments. Candidates: [backlog.md](./backlog.md).

- Pagination / cursors
- Prefs persistence
- LOG-6 / LOG-7 (edit, delete, manual entry)
- Insights aggregation endpoints
- `PATCH /me`
- Multi-user workspaces

---

## Tracking

Use this roadmap as the checklist. Split Phase 1+ into issues later if useful; no issue tracker is required for Phase 0.

---

## Related documents

- [prd.md](./prd.md)
- [backlog.md](./backlog.md)
- [plans/phase-0-planning.md](./plans/phase-0-planning.md)
- [adr/0001-backend-stack.md](./adr/0001-backend-stack.md)
- [adr/0002-separate-repository.md](./adr/0002-separate-repository.md)
