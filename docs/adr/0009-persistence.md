# ADR-0009: Persistence

**Status:** Accepted  
**Date:** 2026-08-14  
**Deciders:** Project owner

## Context

The frontend mock is an in-memory workspace keyed by a request header. Reload reseeds fixtures. That is the opposite of what this repository is for.

We need durable storage for projects, sessions, and a profile so a process restart and a second device see the same data. Stack ([ADR-0001](./0001-backend-stack.md)) is Go. Engine, hosting, and migration tool were still open.

## Decision

An in-memory store is allowed **only** as a test double.

| Topic | Choice |
| --- | --- |
| Engine | PostgreSQL |
| Driver | `pgx` via `database/sql` |
| Queries | `sqlc` generating Go from SQL. No GORM / Ent as the source of truth. |
| Migration tool | goose (plain SQL files) |
| Where it runs | Local Docker Compose (PostgreSQL 16). App reads `DATABASE_URL`. v1 production is the same Compose Postgres on the owner’s machine ([ADR-0011](./0011-local-production-host.md)). |
| IDs | UUID strings. Opaque on the wire; do not require `proj-` / `sess-` prefixes. |

Schema includes an internal `user_id` ([ADR-0006](./0006-single-user-tenancy.md)). That column is not exposed on the wire.

Constraints the choice must satisfy:

1. Process restart does not lose projects or stopped sessions.
2. Domain invariants are enforced in application code (and/or DB constraints that match them). Do not rely on the HTTP layer alone.
3. Wire DTOs stay as specified; column names may differ.
4. Single-user v1, but avoid a process-wide singleton ([ADR-0006](./0006-single-user-tenancy.md)).
5. Backups are possible (Phase 4 / [ADR-0011](./0011-local-production-host.md)): `pg_dump` + a restore drill.

## Consequences

### Positive

- Phase 2 can implement the contract against a real store.
- Production-shaped locally and in CI (Compose + a Postgres service).
- SQL migrations stay reviewable. sqlc keeps queries typed.

### Negative / tradeoffs

- Postgres is more moving parts on a laptop than a SQLite file. Compose + a CI service is the mitigation from Phase 1.
- Tests that need durability require a running Postgres (or a later testcontainer). Domain tests do not.

## Amendment (2026-08-18)

The production host is no longer TBD. [ADR-0011](./0011-local-production-host.md) keeps this Compose Postgres as the system of record on the owner’s machine.

## Alternatives considered

| Option | Why not |
| --- | --- |
| SQLite (file) | Zero ops for single-user; harder multi-instance. Owner chose Postgres for a production-shaped store from day one. |
| In-memory only | Rejected as the system of record. Tests only. |
| Document store | Session/project rules are relational enough; extra novelty. |
| GORM / Ent as source of truth | Hides migrations; easier to drift from the contract. |
| Frontend-only IndexedDB | Rejected by frontend ADR-0002 / 0004; this repo exists so we do not do that. |

## Related

- [0001-backend-stack.md](./0001-backend-stack.md)
- [0006-single-user-tenancy.md](./0006-single-user-tenancy.md)
- [../domain-model.md](../domain-model.md)
