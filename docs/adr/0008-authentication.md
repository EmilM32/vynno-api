# ADR-0008: Authentication

**Status:** Deferred  
**Date:** 2026-08-14  
**Deciders:** Project owner

## Context

The published contract has **no auth**. The SPA login is a stub: any username + password are accepted and stored in `sessionStorage`. `ApiClient` has a single place to attach credentials later.

Frontend Phase 5c (point at this API and delete `/mock/v1`) expects real auth. A public unauthenticated write API is not acceptable for production. Phase 2 (implement the contract locally) can run without it.

This ADR decides the mechanism. It does not add login/register routes to the contract until Accepted and the contract is amended.

## Decision

**Deferred to Phase 3.** No auth on the wire until then. Phase 0–2 ship an unauthenticated local API (same as the mock, minus `X-Mock-Workspace`).

Do not implement auth, CORS credential flags, or login routes until this ADR is Accepted.

| Topic | Choice |
| --- | --- |
| Mechanism | _TBD in Phase 3_ (session cookie, bearer token, …) |
| Credential fields | _TBD in Phase 3_ |
| New routes | _TBD_ — amend [../api-contract.md](../api-contract.md) first |
| CORS / cookie flags | _TBD in Phase 3_ (Gin + `gin-contrib/cors`) |
| What is public | Everything, until Phase 3. After Phase 3: likely nothing. |

Constraints the choice must satisfy (when Accepted):

1. The SPA can attach credentials in `ApiClient` only — no per-view auth code.
2. Unauthenticated writes (and likely reads) fail with a documented code after Phase 3.
3. Single-user v1 ([0006](./0006-single-user-tenancy.md)) — no roles, no invites.
4. No secrets in the frontend repo.

## Consequences

### Positive

- Phase 0 can exit. Phase 1–2 are unblocked.
- Frontend can talk to a local `/v1` the same way it talks to `/mock/v1`.

### Negative / tradeoffs

- A public unauthenticated write API is not acceptable for production. Phase 3 blocks deploy and frontend Phase 5c.
- Cookie vs bearer is still open; CORS work waits with it.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Accept a mechanism now (HTTP-only session cookie) | Simple for a first-party SPA; needs CORS + CSRF. Does not unblock Phase 1–2. Owner deferred. |
| Accept a mechanism now (bearer token) | Easy to attach on `ApiClient`; refresh/storage to design. Owner deferred. |
| Magic link / passwordless | Nice UX; more email infrastructure. Not for v1. |
| OAuth-only (Google, GitHub) | Fine later; heavy for a single-user v1. |
| Keep the stub forever | Rejected for any deployed API. |

## Related

- [../frontend-handoff.md](../frontend-handoff.md)
- [../prd.md](../prd.md) §8.4
- [0006-single-user-tenancy.md](./0006-single-user-tenancy.md)
- [../api-contract.md](../api-contract.md) (Out of scope)
- [../roadmap.md](../roadmap.md) Phase 3
