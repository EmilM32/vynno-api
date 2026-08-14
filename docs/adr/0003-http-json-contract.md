# ADR-0003: HTTP JSON contract is the API

**Status:** Accepted  
**Date:** 2026-08-14  
**Deciders:** Project owner  
**Inherited from:** Frontend ADR-0010 (2026-08-13, amended 2026-08-14), “HTTP JSON contract (DTO-first, mock HTTP)”

## Context

The SPA loads and mutates data through `fetch` against a documented REST contract. Mock `/mock/v1` implements the same paths and DTOs. The remaining frontend swap is a base URL (and auth), not a data-layer rewrite.

If this backend invented GraphQL, RPC, or different field names, the SPA would have to change schemas, mappers, and possibly stores. That is the opposite of why the contract was written.

## Decision

1. **The API is [../api-contract.md](../api-contract.md).** Paths, methods, status codes, DTO shapes, and error codes are not optional.
2. **Prefix `/v1`.** Version in the path, not a custom header.
3. **JSON, camelCase.** Lists are `{ "items": T[] }`. Absent optionals are JSON `null`, not omitted keys.
4. **Error envelope** `{ "error": { "code", "message" } }` on every failure. `code` is from the documented set. `message` is for logs.
5. **Verb routes** for session and project lifecycle (`/pause`, `/resume`, `/stop`, `/archive`, `/restore`). No generic `PATCH status`.
6. **DTO-first.** Persistence may use different column names; handlers map to the DTO. Do not leak SQL or framework types onto the wire.
7. **No SvelteKit remote functions, tRPC, or similar** as the public surface. Those are not what the SPA calls.
8. New resources or fields require a contract amendment **before** implementation.

## Consequences

### Positive

- Frontend Phase 5c is a configuration change plus auth.
- Domain tests can assert on codes and state without HTTP.
- A later OpenAPI file can be generated from this contract; it is not a prerequisite.

### Negative / tradeoffs

- We cannot “improve” field names (`isArchived` vs `archived`) without a paired frontend change.
- Pagination, prefs, and insights endpoints are intentionally missing until amended.
- Full session list on boot will not scale forever ([../open-questions.md](../open-questions.md) #5).

## Alternatives considered

| Option | Why not |
| --- | --- |
| GraphQL | SPA is written against REST resources. |
| RPC / remote functions | Kit-specific; not portable; frontend explicitly rejected this. |
| Different DTO naming to match the DB | Forces a mapper rewrite on the client. |
| JSON:API / HAL | Heavier than `{ items }` and unused by the client. |

## Related

- [../api-contract.md](../api-contract.md)
- [../frontend-handoff.md](../frontend-handoff.md)
- [0002-separate-repository.md](./0002-separate-repository.md)
- [0005-session-lifecycle.md](./0005-session-lifecycle.md)
