# ADR-0008: Authentication

**Status:** Proposed  
**Date:** 2026-08-14  
**Deciders:** Project owner

## Context

The published contract has **no auth**. The SPA login is a stub: any username + password are accepted and stored in `sessionStorage`. `ApiClient` has a single place to attach credentials later.

Frontend Phase 5c (point at this API and delete `/mock/v1`) expects real auth. A public unauthenticated write API is not acceptable for production. Phase 2 (implement the contract locally) can run without it.

This ADR decides the mechanism. It does not add login/register routes to the contract until Accepted and the contract is amended.

## Decision

**Undecided.** Accept this ADR by choosing a mechanism and listing the contract amendments (if any). Do not implement auth until then.

| Topic | Choice |
| --- | --- |
| Mechanism | _TBD_ (session cookie, bearer token, …) |
| Credential fields | _TBD_ (email/password, magic link, …) |
| New routes | _TBD_ — amend [../api-contract.md](../api-contract.md) first |
| CORS / cookie flags | _TBD_ |
| What is public | _TBD_ (likely nothing once auth ships) |

Constraints the choice must satisfy:

1. The SPA can attach credentials in `ApiClient` only — no per-view auth code.
2. Unauthenticated writes (and likely reads) fail with a documented code after Phase 3.
3. Single-user v1 ([0006](./0006-single-user-tenancy.md)) — no roles, no invites.
4. No secrets in the frontend repo.

## Consequences

### Positive (once accepted)

- Frontend can drop the stub login.
- Multi-device becomes meaningful.

### Negative / tradeoffs

- Any cookie vs header choice affects CORS and local HTTPS.
- Leaving this Proposed does **not** block Phase 2. It **does** block production and frontend Phase 5c.

## Alternatives considered

Fill “Why not” when choosing.

| Option | Why not |
| --- | --- |
| HTTP-only session cookie | Simple for a first-party SPA; needs CORS + CSRF story. Not chosen yet. |
| Bearer access token (JWT or opaque) | Easy to attach on `ApiClient`; refresh/storage to design. Not chosen yet. |
| Magic link / passwordless | Nice UX; more email infrastructure. Not chosen yet. |
| OAuth-only (Google, GitHub) | Fine later; heavy for a single-user v1. Not chosen yet. |
| Keep the stub forever | Rejected for any deployed API. |

## Related

- [../frontend-handoff.md](../frontend-handoff.md)
- [../prd.md](../prd.md) §8.4
- [0006-single-user-tenancy.md](./0006-single-user-tenancy.md)
- [../api-contract.md](../api-contract.md) (Out of scope)
