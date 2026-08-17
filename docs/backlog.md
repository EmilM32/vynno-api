# Backlog — Vynno API

**Last updated:** 2026-08-17  
**Context:** Work we are deliberately **not** doing in the current phase. Do not pull these into Phase 3 without a contract amendment and a roadmap change.

IDs that start with frontend requirement codes (`LOG-6`, …) are the same items as in the frontend PRD / P2 backlog. They imply API work only when the SPA is ready to call new endpoints.

---

## Contract extensions (need an amendment first)

| ID | Item | Notes |
| --- | --- | --- |
| LOG-6 | Edit / delete a stopped session | Not in [api-contract.md](./api-contract.md). Would need verbs or PATCH/DELETE on `/sessions/:id`. |
| LOG-7 | Manual time entry | Create a stopped session without running the timer. New body shape. |
| PREFS | Persist daily target and default project | Client `prefsStore` is in-memory. New resource. |
| PAGE | Cursor / offset pagination | v1 loads the full session list. Revisit when history is large. |
| INS | Insights / dashboard aggregate endpoints | Client computes these from sessions. |
| TMR-9 | Session target duration UI | Field already exists on `StartSessionDto`; no API change until the UI ships. |

## Backend-only later

| ID | Item | Notes |
| --- | --- | --- |
| AUTH-EXT | OAuth / passwordless / 2FA / password reset | After ADR-0008’s cookie session works. |
| MULTI | Team workspaces | Contradicts [ADR-0006](./adr/0006-single-user-tenancy.md) until we supersede it. |
| RATE | Rate limiting | Production hardening; not a contract field. |
| OPENAPI | Generated OpenAPI + client | Optional; SPA already has Valibot schemas. |
| WEBHOOK | Outbound webhooks | No product request. |

## Not this product

Invoicing, payroll, calendar sync, IDE plugins, native mobile APIs. Same non-goals as the frontend PRD.

---

## Related

- [prd.md](./prd.md) §4 and §8
- [roadmap.md](./roadmap.md) Phase 5
- [api-contract.md](./api-contract.md) Out of scope
