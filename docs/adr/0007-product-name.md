# ADR-0007: Public product name is Vynno

**Status:** Accepted  
**Date:** 2026-08-14  
**Deciders:** Project owner  
**Inherited from:** Frontend ADR-0009 (2026-08-13)

## Context

The product shipped in mockups and early docs as **DevTime**. The public name is **Vynno** (VIN-oh). The frontend git repo may still be named `dev-time`.

This API repository should use the public name from day one.

## Decision

1. The **public name** is **Vynno**. Say VIN-oh. Spell it with a double *n*. It is not an acronym.
2. **DevTime** is historical. It may appear in inherited context and in references to the frontend repo.
3. New prose, module names, env prefixes, and user-facing API `message` strings say Vynno (or are generic). Do not introduce a `DevTime` identifier in this repo.
4. Tagline remains “Where the hours went.”

## Consequences

### Positive

- This repo does not inherit the frontend’s name split.
- Logs and error messages match the product.

### Negative / tradeoffs

- Cross-links to the frontend repo will still say `dev-time`.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Keep DevTime in the API | The product is not called that. |
| Invent a third service name (`vynno-api` as the product) | The product is Vynno; this is its API. |

## Related

- [../prd.md](../prd.md)
