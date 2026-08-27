# Backlog — Vynno API

**Last updated:** 2026-08-27  
**Context:** Work we are deliberately **not** doing now. Do not pull these in without a contract amendment and a roadmap change.

IDs that start with frontend requirement codes are the same items as in the frontend PRD / P2 backlog. They imply API work only when the SPA is ready to call new endpoints.

---

## Contract extensions (need an amendment first)

| ID | Item | Notes |
| --- | --- | --- |
| PREFS | Persist daily target and default project | Client `prefsStore` is in-memory. New resource. |
| INS | Insights / dashboard aggregate endpoints | Client computes these from sessions. |
| TMR-9 | Session target duration UI | Field already exists on `StartSessionDto`; no API change until the UI ships. |

## Backend-only later

| ID | Item | Notes |
| --- | --- | --- |
| AUTH-EXT | OAuth / passwordless / 2FA / change-email / logged-in change-password | Cookie session, register confirmation, and password reset already shipped. |
| MULTI | Team workspaces | Contradicts [ADR-0006](./adr/0006-single-user-tenancy.md) until we supersede it. |
| RATE | Rate limiting | Production hardening; not a contract field. Revisit before any internet-facing host. |
| OPENAPI | Generated TypeScript / Valibot client | Spec + Swagger UI shipped ([ADR-0013](./adr/0013-openapi-swagger.md)). SPA already has Valibot schemas. |
| WEBHOOK | Outbound webhooks | No product request. |

## Not this product

Invoicing, payroll, calendar sync, IDE plugins, native mobile APIs. Same non-goals as the frontend PRD.

---

## Related

- [prd.md](./prd.md)
- [roadmap.md](./roadmap.md) Phase 5
- [api-contract.md](./api-contract.md) Out of scope
