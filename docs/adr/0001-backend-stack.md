# ADR-0001: Backend stack

**Status:** Proposed  
**Date:** 2026-08-14  
**Deciders:** Project owner

## Context

The API repository does not exist yet. The frontend already speaks a stack-agnostic HTTP JSON contract ([ADR-0003](./0003-http-json-contract.md)). We need a language, HTTP framework, and module layout before Phase 1 scaffold.

This ADR does **not** choose a database ([ADR-0009](./0009-persistence.md)) or an auth mechanism ([ADR-0008](./0008-authentication.md)). Those are separate decisions. Hosting (VPS, Fly, Cloud Run, …) can be a later amendment or its own ADR.

## Decision

**Undecided.** Accept this ADR by filling the table and moving Status to Accepted. Do not scaffold until then.

| Layer | Choice |
| --- | --- |
| Language | _TBD_ |
| HTTP framework | _TBD_ |
| Validation | _TBD_ (must be able to reject bodies with `invalid_body`) |
| Test runner | _TBD_ |
| Lint / format | _TBD_ |

Constraints the choice must satisfy:

1. Implement [../api-contract.md](../api-contract.md) without leaking framework RPC into the wire format.
2. Domain rules live in testable functions/modules, not only in HTTP handlers ([../domain-model.md](../domain-model.md)).
3. No frontend types or SvelteKit remote functions as the API surface.

## Consequences

### Positive (once accepted)

- Scaffold can proceed without re-litigating tools.
- `AGENTS.md` can list real commands and conventions.

### Negative / tradeoffs

- Leaving this Proposed blocks Phase 1. That is intentional.

## Alternatives considered

Fill the “Why not” column when choosing. These are starting options, not a shortlist we are committed to.

| Option | Why not |
| --- | --- |
| TypeScript (Node / Bun) + a small HTTP framework | Same language as the frontend; easy to share DTO examples. Not chosen yet. |
| Go + stdlib or a small router | Single binary, simple deploy. Not chosen yet. |
| Rust | Strong correctness; slower to iterate. Not chosen yet. |
| Python | Fine for an API; weaker default fit for this contract. Not chosen yet. |
| Put the API inside the SvelteKit repo | Rejected by [0002](./0002-separate-repository.md). |

## Related

- [0002-separate-repository.md](./0002-separate-repository.md)
- [0009-persistence.md](./0009-persistence.md)
- [../plans/phase-0-planning.md](../plans/phase-0-planning.md)
- [../prd.md](../prd.md)
