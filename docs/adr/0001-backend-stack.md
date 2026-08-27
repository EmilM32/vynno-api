# ADR-0001: Backend stack

**Status:** Accepted  
**Date:** 2026-08-14  
**Deciders:** Project owner

## Context

The frontend already speaks a stack-agnostic HTTP JSON contract ([ADR-0003](./0003-http-json-contract.md)). This ADR records the language, HTTP framework, and module layout.

This ADR does **not** choose a database ([ADR-0009](./0009-persistence.md)) or an auth mechanism ([ADR-0008](./0008-authentication.md)). Those are separate decisions. Hosting for v1 is the owner’s machine ([ADR-0011](./0011-local-production-host.md)). A public cloud host would be a later amendment.

## Decision

| Layer | Choice |
| --- | --- |
| Language | Go (module `github.com/EmilM32/vynno-api`) |
| HTTP framework | Gin |
| Validation | Hand-written request checks + domain functions. Gin binders may decode JSON; they are not the source of truth for contract or lifecycle errors. |
| Test runner | `go test`, table-driven |
| Lint / format | `gofmt` + `golangci-lint` |

Constraints the choice must satisfy:

1. Implement [../api-contract.md](../api-contract.md) without leaking framework RPC into the wire format.
2. Domain rules live in testable functions/modules, not only in HTTP handlers ([../domain-model.md](../domain-model.md)).
3. No frontend types or SvelteKit remote functions as the API surface.

Module layout:

```
cmd/api/             # main, listen, wiring
cmd/devdata/         # reset/seed against vynno_dev only
internal/httpserver/ # Gin router, handlers, error envelope
internal/domain/     # session + project rules (no HTTP, no SQL)
internal/service/    # use cases; depends on Store and Mailer
internal/store/      # sqlc queries, migrations applied at boot
internal/mail/       # Mailer port (SMTP / log / discard)
internal/config/     # env loading, no secrets in git
```

`internal/domain` must not import Gin or the database driver.

## Consequences

### Positive

- Scaffold can proceed without re-litigating tools.
- `AGENTS.md` can list real commands and conventions.
- Gin starts quickly and later carries CORS, auth, and security as middleware (Phase 3–4), not a bag of `net/http` handlers.

### Negative / tradeoffs

- `gin.Context` is farther from `net/http` than chi. Handlers stay thin so domain tests do not need Gin.
- Gin struct-tag binding is easy to overuse. Contract codes (`invalid_body`, `last_active_project`, pause accounting) stay in domain code.
- CORS / CSRF / JWT come from `gin-contrib` and third-party packages, assembled in Phase 3–4 — not in Phase 1.

## Amendment (2026-08-18)

v1 production host is the owner’s machine: compiled `bin/vynno-api` + Compose Postgres. Decision: [0011](./0011-local-production-host.md).

## Alternatives considered

| Option | Why not |
| --- | --- |
| TypeScript (Node / Bun) + a small HTTP framework | Same language as the frontend; easy to share DTO examples. Owner chose Go. Shared DTO package is rejected ([../prd.md](../prd.md) non-goals). |
| Echo | Official CORS / JWT / CSRF / Secure middleware. Viable. Owner chose Gin for community and start speed. |
| chi or `net/http` only | Thin and close to the stdlib. Too little later support for CORS, auth, and security as product work. |
| Fiber | fasthttp, not `net/http`. Worse ecosystem fit. |
| Rust | Strong correctness; slower to iterate for this contract. |
| Python | Fine for an API; weaker default fit for a small typed JSON contract. |
| Put the API inside the SvelteKit repo | Rejected by [0002](./0002-separate-repository.md). |

## Related

- [0002-separate-repository.md](./0002-separate-repository.md)
- [0009-persistence.md](./0009-persistence.md)
- [../prd.md](../prd.md)
