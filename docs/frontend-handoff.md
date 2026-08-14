# Frontend handoff

**Status:** Draft  
**Last updated:** 2026-08-14

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
- Login is a stub: any non-empty username + password → `sessionStorage`. Nothing is sent to an API.

No view or store rewrite is required to swap origins.

---

## Swap steps (frontend Phase 5c)

1. This API implements [api-contract.md](./api-contract.md).
2. Frontend sets `PUBLIC_API_BASE=https://…/v1`.
3. Frontend adds auth on `ApiClient` in one place (`src/lib/api/client.ts`) — depends on [ADR-0008](./adr/0008-authentication.md).
4. Frontend deletes `src/routes/mock/v1/`, `$lib/api/fixtures/`, and `$lib/api/mock/`.

Until step 3, the SPA can talk to an unauthenticated local API the same way it talks to the mock (minus the `X-Mock-Workspace` header, which is mock-only).

---

## What the client will send

Paths are relative to `/v1`:

| Method | Path |
| --- | --- |
| GET | `/me` |
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

CORS must allow the SPA origin once the API is not same-origin. Cookie vs `Authorization` is part of ADR-0008 — do not pick one in handlers first.

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

---

## Auth today vs later

| Stage | Frontend | This API |
| --- | --- | --- |
| Now (Phase 0–2) | Stub login, no `Authorization` | Contract has no auth |
| Phase 3 | `ApiClient` attaches credentials | ADR-0008 implemented |
| After swap | Stub login removed | Unauthenticated writes rejected |

Do not block Phase 2 (implement the contract) on auth. Do not ship a public API without Phase 3.

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
