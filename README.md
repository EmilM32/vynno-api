# Vynno API

**VIN-oh** — _Where the hours went._

This repository will be the **backend** for Vynno: persistence, auth, and the HTTP JSON API the SvelteKit app already speaks.

The frontend lives in a separate repo (`dev-time`). It currently runs against mock `/mock/v1`. Pointing it at this API is frontend Phase 5c.

## How to use this folder (while it still lives in `dev-time`)

`backend-docs/` is a **copy-ready kit**. It is not application code.

1. Create a new git repository for the API.
2. Copy **everything in this directory** to that repository’s root (`README.md`, `AGENTS.md`, `docs/`).
3. After the copy, delete the “How to use this folder” section from the new README. The rest of this file is the API repo README.
4. Finish Phase 0: accept or explicitly defer the Proposed ADRs (`0001` stack, `0008` auth, `0009` persistence).
5. Do **not** scaffold code until Phase 0 exits. That is how the frontend started.

This frontend repo can keep `backend-docs/` as a snapshot, or replace it with a link to the new repo once that exists.

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
