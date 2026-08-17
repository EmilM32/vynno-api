# Plan — Phase 3 auth

**Status:** Done  
**Last updated:** 2026-08-17  
**Tracking:** Roadmap Phase 3  
**Depends on:** Phase 2 Done, [ADR-0008](../adr/0008-authentication.md) Accepted

---

## Summary

Authenticate `/v1` with an opaque session in an HttpOnly cookie. Login persists across tab and browser restart when remember-me is on (the default). Unauthenticated reads and writes return `401 unauthorized`. Data is scoped to the authenticated `user_id`.

The SPA attaches the cookie in `ApiClient` (`credentials: 'include'`) and drops the stub login. That frontend work is Phase 5c in the `vynno` repo.

## Why now

Phase 2 is an unauthenticated write API. That is not acceptable to deploy, and the SPA cannot leave `/mock/v1` until credentials exist.

## Constraints

- Only documented paths, fields, and error codes. Amend the contract first.
- `internal/domain` imports neither Gin nor the database driver.
- No roles, invites, or workspaces ([ADR-0006](../adr/0006-single-user-tenancy.md)).
- The session secret is never in a JSON body.
- CORS origins are locked to `SPA_ORIGIN` (cookies cannot use `*`).
- No register screen in the SPA this phase.

## Approach

1. Accept [ADR-0008](../adr/0008-authentication.md); amend [../api-contract.md](../api-contract.md).
2. goose `00002_auth.sql`: `users.username`, `users.password_hash`, `auth_tokens`.
3. sqlc + Memory store scoped by `user_id` (tokens, credentials, isolation).
4. Domain: username/password rules; codes `unauthorized`, `invalid_credentials`, `username_in_use`.
5. Service: register, login (bcrypt), logout, resolve token. `ForUser` for request-scoped identity. Stop pinning `FirstUserID` as the process user.
6. Gin: cookie + optional Bearer middleware; CORS credentials + origin allowlist; Origin check on cookie-backed mutations; login/register/logout handlers.
7. Boot: `BOOTSTRAP_USERNAME` / `BOOTSTRAP_PASSWORD` attach creds to the Phase 2 seed user when `password_hash` is null.
8. Frontend 5c (paired): contract + schemas, `credentials: 'include'`, remember-me checkbox, delete mock tree.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Extra JSON fields | SPA `invalid_response` | Response is `{ profile }` only; Valibot updated in the same change set |
| Memory store ignores `userID` | Isolation tests lie | Scope Memory by user before those tests |
| CORS `*` + credentials | Browser drops the cookie | `SPA_ORIGIN` required at boot |
| `Secure` on localhost HTTP | Cookie never stored | `COOKIE_SECURE` defaults false |
| Token in the JSON body | Secret lands in JS | Do not return it |

## Out of scope

JWT, OAuth, magic link, 2FA, password reset, register UI, workspaces, the rest of Phase 4 (observability, backups, deploy).

## Exit checklist

- [x] ADR-0008 Accepted; contract amended in both repos
- [x] Unauthenticated `/v1` reads and writes → `401 unauthorized`
- [x] Login sets `vynno_session`; Timer + project CRUD work (HTTP tests + login e2e)
- [x] Remember-me Max-Age vs session cookie covered in HTTP tests
- [x] Two users do not share projects or sessions
- [x] Logout revokes the server token and clears the cookie
- [x] Frontend stub login gone; mock tree deleted
- [x] Roadmap Phase 3 boxes; this plan **Done**

## Related

- [../roadmap.md](../roadmap.md)
- [../adr/0008-authentication.md](../adr/0008-authentication.md)
- [../api-contract.md](../api-contract.md)
- [../frontend-handoff.md](../frontend-handoff.md)
