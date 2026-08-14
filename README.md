# Vynno API

**VIN-oh** — _Where the hours went._

This repository will be the **backend** for Vynno: persistence, auth, and the HTTP JSON API the SvelteKit app already speaks.

The frontend lives in a separate repo (`dev-time`). It currently runs against mock `/mock/v1`. Pointing it at this API is frontend Phase 5c.

## Scope of this repository

**API only.**

| In this repo | Separate repo (`dev-time`) |
| --- | --- |
| HTTP JSON API under `/v1` | SvelteKit UI, design system, i18n |
| Database and durable writes | HTTP-fetched mock JSON until swap |
| Authentication (Phase 3) | Stub login (`sessionStorage`) until swap |
| Server-side session and project rules | Client display, aggregates, theme, locale |

## Stack

**Not chosen.** Accept [ADR-0001](./docs/adr/0001-backend-stack.md) and [ADR-0009](./docs/adr/0009-persistence.md) before scaffolding.

Until then: no framework, ORM, or cloud vendor in this repo.

## Documentation

Start here: **[docs/README.md](./docs/README.md)**

How we write PRDs, ADRs, and plans: **[docs/working-agreement.md](./docs/working-agreement.md)**

| Doc | Description |
| --- | --- |
| [docs/prd.md](./docs/prd.md) | Product requirements for the API |
| [docs/domain-model.md](./docs/domain-model.md) | Entities and server-enforced rules |
| [docs/api-contract.md](./docs/api-contract.md) | Wire JSON the SPA already speaks |
| [docs/roadmap.md](./docs/roadmap.md) | Delivery phases |
| [docs/adr/](./docs/adr/) | Architecture decisions |
| [docs/plans/](./docs/plans/) | Implementation plans |

## Status

- **Phase 0:** Planning — this kit. Stack, persistence, and auth direction still Proposed.
- **Phase 1+:** Not started. Depends on Phase 0 exit.
