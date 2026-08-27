# Backlog — Vynno API

**Last updated:** 2026-08-27  
**Context:** Work we are deliberately **not** doing now. Phase 4 (local production) is done. Do not pull these in without a contract amendment and a roadmap change.

IDs that start with frontend requirement codes (`LOG-6`, …) are the same items as in the frontend PRD / P2 backlog. They imply API work only when the SPA is ready to call new endpoints.

---

## Contract extensions (need an amendment first)

| ID | Item | Notes |
| --- | --- | --- |
| LOG-6 | Edit / delete a session | **Done** — [plans/session-edit.md](./plans/session-edit.md). `PATCH` / `DELETE /sessions/:id`. |
| LOG-7 | Manual time entry | **Done** — same plan. `POST /sessions/manual`. |
| PREFS | Persist daily target and default project | Client `prefsStore` is in-memory. New resource. |
| PAGE | Cursor / offset pagination | **Done** — [plans/session-pagination.md](./plans/session-pagination.md). Cursor on `GET /sessions`. |
| INS | Insights / dashboard aggregate endpoints | Client computes these from sessions. |
| TMR-9 | Session target duration UI | Field already exists on `StartSessionDto`; no API change until the UI ships. |

## Backend-only later

| ID | Item | Notes |
| --- | --- | --- |
| AUTH-EXT | OAuth / passwordless / 2FA / change-email / logged-in change-password | Cookie session, register confirmation, and password reset shipped ([plans/email.md](./plans/email.md) Done). |
| MULTI | Team workspaces | Contradicts [ADR-0006](./adr/0006-single-user-tenancy.md) until we supersede it. |
| RATE | Rate limiting | Production hardening; not a contract field. |
| OPENAPI | Generated TypeScript / Valibot client | Spec + Swagger UI shipped ([ADR-0013](./adr/0013-openapi-swagger.md)). SPA already has Valibot schemas. |
| WEBHOOK | Outbound webhooks | No product request. |

## Not this product

Invoicing, payroll, calendar sync, IDE plugins, native mobile APIs. Same non-goals as the frontend PRD.

---

## Related

- [prd.md](./prd.md) §4 and §8
- [roadmap.md](./roadmap.md) Phase 5
- [api-contract.md](./api-contract.md) Out of scope
