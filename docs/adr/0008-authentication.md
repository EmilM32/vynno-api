# ADR-0008: Authentication

**Status:** Accepted  
**Date:** 2026-08-14  
**Accepted:** 2026-08-17  
**Deciders:** Project owner

## Context

The published contract had **no auth**. The SPA login was a stub: any username + password were accepted and stored in `sessionStorage`. `ApiClient` has a single place to attach credentials.

Frontend Phase 5c (point at this API and delete `/mock/v1`) expects real auth. A public unauthenticated write API is not acceptable for production. Phase 0–2 shipped without it so the contract could land first.

This ADR decides the mechanism. Login/register routes are in [../api-contract.md](../api-contract.md).

## Decision

1. **Opaque session token** stored as SHA-256 on the server (`auth_tokens`). Never JWT.
2. **Transport for the SPA:** HttpOnly cookie `vynno_session`. The JSON body of login/register is `{ profile }` only — the secret is not returned to JavaScript.
3. **Secondary transport:** `Authorization: Bearer <token>` for tests, curl, and non-browser clients. The SPA does not use this.
4. **Remember me:** `rememberMe` on login/register (boolean, default **true**). True → cookie `Max-Age` 30 days. False → session cookie (cleared when the browser quits). Server `expires_at` is always 30 days.
5. **Credentials:** `username` + `password`. Password hashed with bcrypt. Username is `^[a-z0-9_]{3,32}$` after trim+lowercase. (**Amended 2026-08-26:** `email` + `password`. See amendment.)
6. **Public routes:** `GET /healthz`, `POST /v1/auth/login`, `POST /v1/auth/register`, `POST /v1/auth/register/code`, `POST /v1/auth/password/forgot`, `POST /v1/auth/password/reset`, `GET /v1/avatars/:id`. Every other `/v1` resource requires a valid session (reads and writes).
7. **Accounts:** many personal accounts, isolated by internal `user_id`. No roles, invites, or workspaces ([0006](./0006-single-user-tenancy.md)).
8. **CORS:** lock to `SPA_ORIGIN` (comma-separated) plus `PUBLIC_API_ORIGIN` (same-origin Swagger UI). `AllowCredentials: true`. `*` is incompatible with cookies.
9. **CSRF:** `SameSite=Lax` + JSON `Content-Type` on mutating requests + reject mutating cookie requests whose `Origin` (or `Referer` if `Origin` is absent) is not in `SPA_ORIGIN` **or** `PUBLIC_API_ORIGIN`. Bearer-only requests skip the Origin check. The API origin is trusted so same-origin Swagger UI Try-it-out can send the session cookie ([0013](./0013-openapi-swagger.md)).
10. **Cookie flags:** `HttpOnly`; `SameSite=Lax`; `Path=/`; `Secure` when `COOKIE_SECURE=true`.
11. **SPA attach point:** `ApiClient` sets `credentials: 'include'`. It does not store the session secret. An email/logged-in flag may live in `localStorage` (remember-me) or `sessionStorage` (not remembered) so the UI can skip `/login`; a `401 unauthorized` clears it.

| Topic | Choice |
| --- | --- |
| Mechanism | HttpOnly cookie `vynno_session` (opaque token) |
| Credential fields | `email`, `password`, optional `rememberMe` |
| New routes | `POST /v1/auth/register`, `POST /v1/auth/register/code`, `POST /v1/auth/login`, `POST /v1/auth/logout`, `POST /v1/auth/password/forgot`, `POST /v1/auth/password/reset` |
| CORS / cookie flags | `SPA_ORIGIN` allowlist, credentials on, flags in §9–10 |
| What is public | Health + login + register + register code + password forgot/reset + `GET /v1/avatars/:id`. Everything else under `/v1` is authenticated. |

Constraints still in force:

1. The SPA attaches credentials in `ApiClient` only — no per-view auth headers.
2. Unauthenticated reads and writes fail with a documented code (`unauthorized`).
3. Single-user product — no roles, no invites.
4. No secrets in the frontend repo.

## Consequences

### Positive

- Closing the tab or browser does not sign the user out when remember-me is on (the default).
- XSS cannot read the session secret.
- Logout revokes the server row and clears the cookie.
- Two accounts do not share projects or sessions.
- curl/tests can still send `Authorization: Bearer`.

### Negative / tradeoffs

- CORS must name the SPA origin in Phase 3 (not Phase 4).
- A small CSRF story (Origin check) is required because cookies are sent automatically.
- The SPA cannot read the cookie; it infers “signed in” from a non-secret cache plus `GET /me` / hydrate. Expired cookies surface as `401`.
- Session cookies (`rememberMe: false`) still die when the browser quits — that is the shared-computer escape hatch.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Bearer token in `sessionStorage` | Logs the user out when the tab closes. Rejected for product UX. |
| Bearer token in `localStorage` | Persists, but XSS steals a 30-day credential. Worse than HttpOnly once we persist. |
| JWT access + refresh | Secret rotation and refresh surface we do not need for v1. |
| Magic link / passwordless | Email infrastructure and a click-through URL. v1 proves the mailbox with a 6-digit code instead ([0015](./0015-outbound-email.md)). |
| OAuth-only (Google, GitHub) | Fine later; heavy for a single-user v1. |
| Keep the stub forever | Rejected for any deployed API. |

## Amendment (2026-08-17)

`GET /v1/avatars/:id` is public. Cross-origin `<img src>` from the SPA cannot send the `SameSite=Lax` session cookie, so a cookie-gated image URL would never load. The path uses an unguessable UUID. Decision and [ADR-0010](./0010-avatar-storage.md).

## Amendment (2026-08-21)

CSRF allowlist includes `PUBLIC_API_ORIGIN` in addition to `SPA_ORIGIN`, so operator Swagger UI served from this process can mutate cookie-backed routes. The Gin CORS middleware also lists `PUBLIC_API_ORIGIN` so it does not 403 same-origin Try-it-out POSTs (browsers send `Origin` on those). Do not put the API origin in the `SPA_ORIGIN` env var (that list is the SvelteKit app). Open `/swagger/` at `PUBLIC_API_ORIGIN`, not a hostname variant (`localhost` vs `127.0.0.1`).

## Amendment (2026-08-26)

Credentials are **`email` + `password`**. Email is trim + lowercase, 3–254 characters, parsed by `net/mail.ParseAddress` with the parsed address equal to the whole string (reject `"Name <a@b.c>"`). The domain part must contain a `.`. Unique among accounts. Duplicate register is `409 email_in_use` (replaces `username_in_use`).

Register no longer takes `username`. Profile `handle` (`@` + username) is removed. `ProfileDto` includes `email`. Omitted display name is stored empty; the SPA shows the email as the identity label. Email is not writable after register. No mail is sent (verification and password reset stay backlog AUTH-EXT). **Amended later the same day:** register confirmation and password reset. See the following amendment.

Existing non-email usernames migrate to `{old}@vynno.local`.

## Amendment (2026-08-26) — register confirmation and password reset

Outbound mail: [0015-outbound-email.md](./0015-outbound-email.md).

1. **Register is confirm-before-create.** `POST /v1/auth/register/code` `{ email }` sends a 6-digit code (15 min TTL) when the address is free. `POST /v1/auth/register` requires that `code`. The user row is created only after the code is accepted. Success still sets `vynno_session` and returns `{ profile }`. Taken email on send is `409 email_in_use`. Wrong or expired code is `401 invalid_code`.
2. **Password reset, not reminder.** `POST /v1/auth/password/forgot` `{ email }` always returns `204` for a well-formed address and sends a code only if the account exists. `POST /v1/auth/password/reset` `{ email, code, password }` sets a new bcrypt hash and **deletes every `auth_tokens` row for that user**. No session cookie. The user logs in afterwards.
3. **Codes are hashed at rest** (SHA-256), one active challenge per email+purpose, resend replaces the previous code. Cooldown 60 seconds; 5 sends / hour; 5 guesses then the challenge is spent (`429 rate_limited`). The code is never in JSON. `smtp` mode must not log it.
4. **Existing accounts are not re-verified.** Operator seed/reset still inserts users without mail. Email remains not writable after register. Magic links, OAuth, 2FA, and change-email stay out.

Public-route list and the routes table in the Decision section are updated in place so a reader of the current decision sees the live set. History of the previous public-route list is this amendment.

## Related

- [../prd.md](../prd.md)
- [0006-single-user-tenancy.md](./0006-single-user-tenancy.md)
- [../api-contract.md](../api-contract.md)
- [../domain-model.md](../domain-model.md)
- [0013-openapi-swagger.md](./0013-openapi-swagger.md)
- [0015-outbound-email.md](./0015-outbound-email.md)
