# ADR-0013: OpenAPI spec generated from Gin routes

**Status:** Accepted  
**Date:** 2026-08-21  
**Deciders:** Project owner

## Context

The SPA contract is [../api-contract.md](../api-contract.md). Operators still need a browser UI that lists every live route, DTO, error code, and auth scheme, with Try-it-out.

[0003-http-json-contract.md](./0003-http-json-contract.md) already said a later OpenAPI file can be derived from the contract; it is not a prerequisite and not a generated frontend client (PRD non-goal; backlog `OPENAPI` was “spec + client”).

A hand-written `openapi.yaml` would drift from the handlers. Gin handlers are `func(*gin.Context)`, so the compiler cannot see request or response types from the function signature. Generation has to attach metadata at the **same call that mounts the route**.

## Decision

1. **Generate OpenAPI 3.0 at process start** from the `Server.route(...)` wrappers in `internal/httpserver`. Those wrappers both register the Gin handler and record summary, tags, body/success DTO values, query params, and error codes.
2. **Schemas come from the same DTO structs the handlers already decode and encode** (`json` tags). Validation stays hand-written; the spec does not become the validator.
3. **Serve operator docs outside `/v1`:** `GET /swagger/` (embedded Swagger UI) and `GET /openapi.json` (generated spec). Same pattern as `/healthz`. Not SPA contract resources.
4. **Do not use swaggo** (`// @Summary` comments, generated `docs/` package). Product docs already live in `docs/`. Comments next to a raw `r.GET` can be forgotten; a wrapper cannot.
5. **Do not generate a TypeScript / Valibot client.** The SPA keeps its schemas. A later generated client would be a new ADR.
6. **A test fails if Gin’s route table and the generated spec disagree** (ignoring `/swagger/*` and `/openapi.json`). Adding a handler without `route()` cannot ship.
7. **Contract amendments still start in [../api-contract.md](../api-contract.md).** The OpenAPI metadata on `route()` is updated in the same change so the UI matches the wire.

CSRF for Try-it-out: mutating cookie requests from the API’s own origin are allowed. Amendment on [0008-authentication.md](./0008-authentication.md).

## Consequences

### Positive

- Path inventory cannot silently diverge from the running server.
- DTO field names in the UI come from the structs that hit the wire.
- No second YAML source of truth. No collision with product `docs/`.
- Same-origin Swagger UI can login and send `vynno_session`.

### Negative / tradeoffs

- Descriptions, tags, and which error codes an operation lists are still declared next to the route. They can be incomplete even when the path exists. The inventory test does not check prose quality.
- PATCH present-vs-absent JSON is richer than a static schema; the spec documents optional/nullable fields and the contract sentence stays the authority.
- Embedded Swagger UI increases binary size.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Hand-written `openapi.yaml` | Drifts from handlers unless a human remembers. Rejected after review. |
| swaggo annotations | Swagger 2.0-first; generated `docs/` fights product docs; comments are optional next to `r.GET`. |
| Reflect only `engine.Routes()` | Paths only — no DTOs, errors, query params, or auth. |
| Huma / code-first framework | Replaces Gin ([0001-backend-stack.md](./0001-backend-stack.md)). |
| Scalar / Redoc | Viewer choice; the generated JSON works in those later. Ship Swagger UI now. |
| OpenAPI 3.1 / 3.2 | Swagger UI is strongest on 3.0.x; we do not need 3.1 features. |

## Related

- [../api-contract.md](../api-contract.md)
- [0003-http-json-contract.md](./0003-http-json-contract.md)
- [0008-authentication.md](./0008-authentication.md)
- [../plans/swagger.md](../plans/swagger.md)
- [../backlog.md](../backlog.md) `OPENAPI`
