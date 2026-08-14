# Vynno API — agent instructions

## Project

Vynno (formerly DevTime) is a focus-time tracker. **This repository is the backend** — HTTP API, persistence, and (later) auth.

The SvelteKit frontend is a **separate repository**. It already speaks the contract in [docs/api-contract.md](./docs/api-contract.md). Do not invent a parallel API.

| In this repo | Separate (frontend) |
| --- | --- |
| `/v1` JSON API, database, auth | SvelteKit UI, routing, design system |
| Server-side session and project rules | Display, client aggregates, theme, locale |

### Working rules

- Read [docs/README.md](./docs/README.md) and [docs/working-agreement.md](./docs/working-agreement.md) before changing product or architecture.
- Docs-first: update PRD / ADR / plan / contract **before** or **with** the code they describe. Large work gets a plan under `docs/plans/`.
- Do **not** add endpoints, fields, query params, or error codes that are not in [docs/api-contract.md](./docs/api-contract.md). Amend the contract first (and the frontend schemas once both repos exist).
- Do **not** pick a language, framework, database, or host unless [ADR-0001](./docs/adr/0001-backend-stack.md) / [ADR-0009](./docs/adr/0009-persistence.md) are Accepted (or the user explicitly decides them).
- Do **not** implement auth until [ADR-0008](./docs/adr/0008-authentication.md) is Accepted.
- Enforce the domain rules in [docs/domain-model.md](./docs/domain-model.md) on the server. The SPA already assumes them.
- Single-user product for v1 ([ADR-0006](./docs/adr/0006-single-user-tenancy.md)). No team workspaces.

### Stack conventions

Fill this section when ADR-0001 is Accepted. Until then there is no application code.

### Useful commands

Fill this section at scaffold (Phase 1). Typical set: install, dev server, test, lint, migrate.

### Docs

Product and architecture: `docs/README.md`. ADRs: `docs/adr/`. Plans: `docs/plans/`.
