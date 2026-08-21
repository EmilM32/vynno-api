# Frontend handoff

**Status:** Draft  
**Last updated:** 2026-08-21

How the SvelteKit app attaches to this API. This is not a second contract — [api-contract.md](./api-contract.md) is the wire format.

The frontend repository is [`vynno`](https://github.com/EmilM32/vynno). Frontend **Phase 5c** (live API + cookie auth, mock tree deleted) is done.

---

## What the SPA already does

```
UI  →  SessionStore  →  HttpTimeTrackingRepository  →  fetch  →  /v1 (this API)
```

- Every read and write goes through `HttpTimeTrackingRepository`.
- Base URL is `PUBLIC_API_BASE`. Local default is `/v1` (Kit proxies to this process). An absolute value is the origin **including** `/v1`, e.g. `http://localhost:8080/v1`.
- Request/response bodies are validated with Valibot schemas in `src/lib/api/schemas/`. Those schemas are the **client executable source of truth** until this repo publishes its own.
- Known `error.code` values map to Paraglide strings. Raw `error.message` is for logs / DevTools.
- Login talks to `POST /v1/auth/login`. The session lives in the HttpOnly cookie `vynno_session`. The SPA may cache the username in `localStorage` (remember-me) or `sessionStorage` (not remembered) so chrome can skip `/login`. It never stores the session secret.

No view or store rewrite is required to swap origins beyond `ApiClient` credentials and the login call.

---

## Swap (frontend Phase 5c) — done

The SPA already:

1. Implements [api-contract.md](./api-contract.md) against this origin.
2. Uses `PUBLIC_API_BASE=/v1` (or an absolute `…/v1`).
3. Sets `credentials: 'include'` in `ApiClient`. It does not send `Authorization`.
4. Posts `{ username, password, rememberMe }` on login. “Remember me” is checked by default.
5. Treats `401 unauthorized` as sign-out (clears the username cache, redirects to `/login`).
6. Has deleted `src/routes/mock/v1/`, `$lib/api/fixtures/`, and `$lib/api/mock/`.

The API process must list the SPA origin in `SPA_ORIGIN` (comma-separated). Cookies will not be stored if CORS is `*`. On the owner’s machine that is the Vite origin (`:5173`), the Playwright preview (`:4173`), and the local production Node server (`http://localhost:3000`); see [ADR-0011](./adr/0011-local-production-host.md) and [local-production.md](./local-production.md).

A fresh production database has no users. First daily login is the SPA **register** tab (`POST /v1/auth/register`), not a bootstrap account. Seed users (`alexdev` / `maya` / `rio`) exist only on `vynno_dev`.

Playwright still defaults to `API_ORIGIN` (`:8080`). While the daily binary is on that port, set `E2E_API_BASE=http://localhost:8081/v1` and run `vynno-api` `scripts/dev`, or e2e will register throwaway users into production. Do not change committed `API_ORIGIN` — that is the production BFF.

---

## What the client will send

Paths are relative to `/v1`:

| Method | Path |
| --- | --- |
| POST | `/auth/login` , `/auth/register` , `/auth/logout` |
| GET | `/me` |
| PATCH | `/me` |
| PUT | `/me/avatar` (multipart field `file`) |
| DELETE | `/me/avatar` |
| GET | `/avatars/:id` (public; no cookie) |
| GET | `/projects` , `/projects?includeArchived=true` |
| GET | `/projects/:id` |
| POST | `/projects` |
| PATCH | `/projects/:id` |
| POST | `/projects/:id/archive` , `/projects/:id/restore` |
| DELETE | `/projects/:id` |
| GET | `/projects/:id/session-count` |
| GET | `/activity-types` |
| GET | `/activity-types/:id` |
| POST | `/activity-types` |
| PATCH | `/activity-types/:id` |
| DELETE | `/activity-types/:id` |
| GET | `/activity-types/:id/session-count` |
| GET | `/sessions` , `/sessions?status=active,paused&limit=n` |
| GET | `/sessions/active` |
| GET | `/sessions/:id` |
| POST | `/sessions` |
| POST | `/sessions/manual` |
| PATCH | `/sessions/:id` |
| DELETE | `/sessions/:id` |
| POST | `/sessions/:id/pause` , `/resume` , `/stop` |

Creates expect **201**. Other successful writes **200** with the resource. `DELETE` **204** empty body.

CORS allows the SPA origin and credentials. Cookie flags: [ADR-0008](./adr/0008-authentication.md).

---

## What the client will **not** call

Do not add these to “complete” the API. They are client-local or deferred.

| Concern | Where it lives today |
| --- | --- |
| Theme (`dark` / `light` / `deep-dark`) | `localStorage` |
| Locale (en / pl) | `localStorage` / cookie, Paraglide |
| Daily hour target, default project | In-memory `prefsStore` |
| Insights KPIs, donut, weekly bars | Computed from the session list |
| Command palette, nav chrome | Pure UI |
| `X-Mock-Workspace` | Mock isolation only; never send to this API |
| Forgot-password UI | Later screen |

---

## Auth today

| Stage | Frontend | This API |
| --- | --- | --- |
| Phase 0–2 (historical) | Stub login, no credentials | Contract had no auth |
| Phase 3 / 5c (now) | `credentials: 'include'`; login POST; remember-me; mock gone | ADR-0008 implemented; unauthenticated `/v1` rejected |

---

## Drift

If a live response fails the frontend schema, the SPA surfaces `invalid_response` and the user sees a generic error. That is a **contract bug**, not a UI bug.

Fix order:

1. Decide which side is wrong.
2. Amend [api-contract.md](./api-contract.md) here **and** the frontend `docs/api-contract.md` + `src/lib/api/schemas`.
3. Then change the server (or the mapper).

See [working-agreement.md](./working-agreement.md) §6.

---

## Related

- [api-contract.md](./api-contract.md)
- [adr/0003-http-json-contract.md](./adr/0003-http-json-contract.md)
- [adr/0008-authentication.md](./adr/0008-authentication.md)
- [roadmap.md](./roadmap.md)
