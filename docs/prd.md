# Product Requirements Document — Vynno API

**Status:** Accepted  
**Last updated:** 2026-08-27  
**Product name:** Vynno (formerly DevTime)  
**Repository scope:** Backend only (HTTP API, persistence, auth)

The frontend PRD remains the product-facing document for screens and UX. This PRD covers what the **API repository** must do so the product works with real data.

---

## 1. Vision

Vynno helps a person track where their working time goes — by project, task, and activity type — with a dense, technical UI.

This repository persists those same `/v1` requests the SPA already speaks, survives reload, and follows the user across devices.

## 2. Goals

1. Persist projects, sessions, and a profile so reload and a second device see the same data.
2. Enforce session and project lifecycle on the server (the client cannot be trusted).
3. Expose the `/v1` JSON contract so the SPA can call this origin without a parallel API.
4. Real authentication (cookie session, register confirmation, password reset).

Wire format: [api-contract.md](./api-contract.md). Server rules: [domain-model.md](./domain-model.md).

## 3. Non-goals (this repo / near term)

| Non-goal | Rationale |
| --- | --- |
| Any UI | Frontend repository |
| Team / multi-user workspaces | Single-user product first ([ADR-0006](./adr/0006-single-user-tenancy.md)) |
| Invoicing, payroll, client portals | Out of product scope for v1 |
| Calendar / IDE / Git integrations | Not in the frontend mockups |
| Insights / dashboard aggregation endpoints | Client computes these from the session list |
| Theme, locale, command palette | Device-local on the client |
| Inventing resources not in the contract | Amend the contract first |
| Shared DTO package with the frontend | Dual docs until both sides extract one |
| OpenAPI-generated TypeScript / Valibot client | SPA already has Valibot schemas ([ADR-0013](./adr/0013-openapi-swagger.md)) |

## 4. Assumptions

1. Product name is **Vynno** ([ADR-0007](./adr/0007-product-name.md)).
2. Single concurrent live session (running or paused) per user.
3. Single-user product for v1 ([ADR-0006](./adr/0006-single-user-tenancy.md)).
4. Activity types are a per-user dictionary ([ADR-0012](./adr/0012-activity-types.md)). Empty on register.
5. Insights, prefs, theme, and locale stay on the client until the contract says otherwise.
6. Project `color` is any `#rrggbb`. Restricting to the SPA palette is a UI concern unless [ADR-0004](./adr/0004-project-lifecycle.md) is amended.
7. IDs are opaque strings; do not require `proj-` / `sess-` prefixes.
8. Time is UTC ISO-8601 on the wire; display timezone is the client.

## 5. Later

Candidates: [backlog.md](./backlog.md). Only via contract amendments.

---

## Related

- [domain-model.md](./domain-model.md)
- [api-contract.md](./api-contract.md)
- [roadmap.md](./roadmap.md)
- [working-agreement.md](./working-agreement.md)
- [adr/](./adr/)
