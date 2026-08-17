# Frontend handoff

**Status:** Draft  
**Last updated:** 2026-08-17

How the SvelteKit app will attach to this API. This is not a second contract — [api-contract.md](./api-contract.md) is the wire format.

The frontend repository is [`vynno`](https://github.com/EmilM32/vynno). After this API exists, frontend **Phase 5c** is the swap.

---

## What the SPA already does

```
UI  →  SessionStore  →  HttpTimeTrackingRepository  →  fetch  →  /mock/v1 or this API
```

- Every read and write goes through `HttpTimeTrackingRepository`.
- Base URL is `PUBLIC_API_BASE` (default `/mock/v1`). Live value is the origin **including** `/v1`, e.g. `https://api.example.com/v1`.
- Request/response bodies are validated with Valibot schemas in `src/lib/api/schemas/`. Those schemas are the **client executable source of truth** until this repo publishes its own.
- Known `error.code` values map to Paraglide strings. Raw `error.message` is for logs / DevTools.
- Login talks to `POST /v1/auth/login`. The session lives in the HttpOnly cookie `vynno_session`. The SPA may cache the username in `localStorage` (remember-me) or `sessionStorage` (not remembered) so chrome can skip `/login`. It never stores the session secret.

No view or store rewrite is required to swap origins beyond `ApiClient` credentials and the login call.

---

## Swap steps (frontend Phase 5c)

1. This API implements [api-contract.md](./api-contract.md).
2. Frontend sets `PUBLIC_API_BASE=https://…/v1`.
3. Frontend `ApiClient` sets `credentials: 'include'` in one place (`src/lib/api/client.ts`). Do not send `Authorization`.
4. Login form posts `{ username, password, rememberMe }`. “Remember me” is checked by default.
5. `401 unauthorized` clears the username cache and redirects to `/login`.
6. Frontend deletes `src/routes/mock/v1/`, `$lib/api/fixtures/`, and `$lib/api/mock/`.

The API process must list this SPA origin in `SPA_ORIGIN` (comma-separated). Cookies will not be stored if CORS is `*`.

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
| GET | `/sessions` , `/sessions?status=active,paused&limit=n` |
| GET | `/sessions/active` |
| GET | `/sessions/:id` |
| POST | `/sessions` |
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
| Register / forgot-password UI | Later screen; `POST /auth/register` exists for tests and curl |

---

## Auth today vs later

| Stage | Frontend | This API |
| --- | --- | --- |
| Phase 0–2 | Stub login, no credentials | Contract had no auth |
| Phase 3 | `credentials: 'include'`; login POST; remember-me | ADR-0008 implemented |
| After swap | Stub login and mock tree removed | Unauthenticated `/v1` rejected |

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
