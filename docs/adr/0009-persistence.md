# ADR-0009: Persistence

**Status:** Proposed  
**Date:** 2026-08-14  
**Deciders:** Project owner

## Context

The frontend mock is an in-memory workspace keyed by a request header. Reload reseeds fixtures. That is the opposite of what this repository is for.

We need durable storage for projects, sessions, and a profile so a process restart and a second device see the same data. Engine, hosting, and migration tool are still open. Stack ([ADR-0001](./0001-backend-stack.md)) may constrain the client library, but the data rules do not.

## Decision

**Undecided.** Accept this ADR by filling the table. Phase 2 cannot ship without it. An in-memory store is allowed **only** as a test double.

| Topic | Choice |
| --- | --- |
| Engine | _TBD_ (PostgreSQL, SQLite, …) |
| Migration tool | _TBD_ |
| Where it runs | _TBD_ (local file, managed DB, …) |

Constraints the choice must satisfy:

1. Process restart does not lose projects or stopped sessions ([../prd.md](../prd.md) PLAT-1).
2. Domain invariants are enforced in application code (and/or DB constraints that match them). Do not rely on the HTTP layer alone.
3. Wire DTOs stay as specified; column names may differ.
4. Single-user v1, but avoid a process-wide singleton ([ADR-0006](./0006-single-user-tenancy.md)).
5. Backups are possible (Phase 4).

## Consequences

### Positive (once accepted)

- Phase 2 can implement the contract against a real store.
- Tests can use a disposable database or a faithful fake.

### Negative / tradeoffs

- SQLite is simpler to run locally and weaker for multi-instance later.
- Postgres is more moving parts on a laptop.
- Leaving this Proposed blocks a durable Phase 2.

## Alternatives considered

Fill “Why not” when choosing.

| Option | Why not |
| --- | --- |
| PostgreSQL | Standard for a small API; needs a running server. Not chosen yet. |
| SQLite (file) | Zero ops for single-user; harder multi-instance. Not chosen yet. |
| In-memory only | Rejected as the system of record. Tests only. |
| Document store | Session/project rules are relational enough; extra novelty. Not chosen yet. |
| Frontend-only IndexedDB | Rejected by frontend ADR-0002 / 0004; this repo exists so we do not do that. |

## Related

- [0001-backend-stack.md](./0001-backend-stack.md)
- [0006-single-user-tenancy.md](./0006-single-user-tenancy.md)
- [../domain-model.md](../domain-model.md)
- [../plans/phase-0-planning.md](../plans/phase-0-planning.md)
