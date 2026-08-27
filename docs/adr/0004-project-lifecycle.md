# ADR-0004: Project lifecycle (archive + optional hard delete)

**Status:** Accepted  
**Date:** 2026-08-14  
**Deciders:** Project owner  
**Inherited from:** Frontend ADR-0006 (2026-08-12)

## Context

Sessions always reference a `projectId`. The SPA has a `/projects` management UI (create, edit, archive, restore, hard delete) and pickers that must not show archived projects. Historical logs still need a name and color after archive.

The frontend mock already enforces these rules. The API must enforce them too; the client is not the system of record.

## Decision

1. **Primary remove path is archive** (`archived = true`). Archived projects are excluded from default `GET /projects` but remain available via `GET /projects/:id`.
2. **Hard delete** permanently removes a project only when **zero** sessions reference it. Otherwise `409 project_has_sessions`.
3. **Last active project** cannot be archived or hard-deleted (`409 last_active_project`).
4. **Code uniqueness** applies only when `code` is non-empty (case-insensitive among all non-deleted projects). Normalize to uppercase. Pattern `^[A-Z0-9-]{1,8}$`. `code: null` on PATCH clears it.
5. **Name** is required, trimmed, 1–80 characters.
6. **Color** is `#rrggbb`. Restricting to the SPA palette is a UI concern unless we later amend this ADR.
7. **Start session** on a missing project is `404 not_found`; on an archived project is `409 project_archived`.
8. Archive/restore use **verb routes**, not `PATCH { archived }`. Wrong-state verbs are `409 invalid_transition`.

## Consequences

### Positive

- Historical sessions keep a resolvable project identity after archive.
- Timer pickers stay clean (`GET /projects` without `includeArchived`).
- Matches the mock the SPA is already tested against.

### Negative / tradeoffs

- Insights that join only on the default project list may under-show archived series (same tradeoff as the frontend).
- Hard delete is permanent; no trash beyond archive.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Hard delete only | Breaks or orphans historical logs unless cascade/reassign exists. |
| Archive only | Simpler, but unused projects cannot be purged. |
| Reassign sessions on delete | Higher scope; not in the contract. |
| Soft-delete flag distinct from archive | Two hidden states; the UI has one. |

## Related

- [../domain-model.md](../domain-model.md)
- [../api-contract.md](../api-contract.md)
- [../prd.md](../prd.md)
- [0005-session-lifecycle.md](./0005-session-lifecycle.md)
