# Vynno API Documentation

Planning and architecture docs for the **Vynno backend**. This repository implements the HTTP API, persistence, and (later) authentication. The SvelteKit UI lives in a separate frontend repository ([`vynno`](https://github.com/EmilM32/vynno)).

Brand: say **VIN-oh**. Public name is Vynno ([ADR-0007](./adr/0007-product-name.md)).

## Contents

| Document | Purpose |
| --- | --- |
| [working-agreement.md](./working-agreement.md) | How we write PRD, ADRs, roadmaps, and plans |
| [prd.md](./prd.md) | Product requirements, goals, priorities, out of scope |
| [domain-model.md](./domain-model.md) | Entities, relationships, server-enforced rules |
| [api-contract.md](./api-contract.md) | REST + JSON DTO contract the SPA already speaks |
| [frontend-handoff.md](./frontend-handoff.md) | How the frontend swaps from `/mock/v1` to this API |
| [roadmap.md](./roadmap.md) | Phased backend delivery plan |
| [backlog.md](./backlog.md) | Deferred work (do not pull into the current phase) |
| [open-questions.md](./open-questions.md) | Undecided items with defaults |
| [adr/](./adr/) | Architecture Decision Records |
| [plans/](./plans/) | Implementation plans for a phase or feature |

## Stack (not decided)

Language, framework, and database are **Proposed**. See [ADR-0001](./adr/0001-backend-stack.md) and [ADR-0009](./adr/0009-persistence.md). Do not scaffold until Phase 0 exits.

What is already decided:

- Separate repository from the frontend — [ADR-0002](./adr/0002-separate-repository.md)
- HTTP JSON contract (`/v1`) — [ADR-0003](./adr/0003-http-json-contract.md)
- Project and session rules — [ADR-0004](./adr/0004-project-lifecycle.md), [ADR-0005](./adr/0005-session-lifecycle.md)
- Single-user v1 — [ADR-0006](./adr/0006-single-user-tenancy.md)

## Status

Phase 0 (planning) is in progress. This kit is the Phase 0 deliverable. Next: accept Proposed ADRs, then scaffold.

See [roadmap.md](./roadmap.md) and [plans/phase-0-planning.md](./plans/phase-0-planning.md).
