# ADR-0006: Single-user tenancy for v1

**Status:** Accepted  
**Date:** 2026-08-14  
**Deciders:** Project owner  
**Inherited from:** Frontend PRD assumptions (single-user product)

## Context

The frontend is a personal focus timer. There are no team workspaces, sharing, or roles in the mockups or PRD. Adding `ownerId` / `workspaceId` everywhere now would be speculative schema, and the contract has no such fields.

Auth will still exist (Phase 3) so that *a* user can sign in. That is not multi-tenancy.

## Decision

1. **v1 is one person per deployment / account.** All projects and sessions belong to that user.
2. **No workspace, team, member, or sharing resources** until the product asks for them.
3. Persistence may still have a `user_id` column once auth exists, so a later multi-user step does not require a rewrite. It is not exposed on the wire in v1.
4. “One live session” is **per user**, not global to the process. Even in single-user v1, do not use a process-wide singleton that would break two instances or a later second user.

## Consequences

### Positive

- Schema and handlers stay small.
- Matches the SPA and the product.
- A nullable/internal `user_id` later does not change DTOs.

### Negative / tradeoffs

- Two people cannot share a project list.
- If we skip an internal user key entirely, adding auth later is a migration.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Multi-tenant workspaces now | No product requirement; invents API surface. |
| Anonymous global dataset, no user key ever | Collides with Phase 3 auth and any second account. |

## Related

- [../prd.md](../prd.md)
- [0008-authentication.md](./0008-authentication.md)
- [0009-persistence.md](./0009-persistence.md)
