# Roadmap — Vynno API

**Status:** Accepted  
**Last updated:** 2026-08-27  
**Scope:** This repository only (API). Frontend is a separate project.

---

## Shipped

Phases 0–4: planning, scaffold, `/v1` contract, cookie auth, local production (binary + Compose Postgres).

After Phase 4: profile/avatar, playground seed/reset (`vynno_dev`), user-defined activity types, session edit/delete/manual entry, cursor pagination on `GET /sessions`, email login identifier, outbound mail (register confirmation + password reset), operator Swagger UI.

## Later (Phase 5)

Only via contract amendments. Candidates: [backlog.md](./backlog.md).

- Prefs persistence
- Insights aggregation endpoints
- Multi-user workspaces
- OAuth / 2FA / change-email / logged-in change-password (AUTH-EXT remainder)

---

## Related

- [prd.md](./prd.md)
- [backlog.md](./backlog.md)
- [adr/](./adr/)
