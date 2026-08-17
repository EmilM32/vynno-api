# Vynno API

**VIN-oh** — _Where the hours went._

This repository will be the **backend** for Vynno: persistence, auth, and the HTTP JSON API the SvelteKit app already speaks.

The frontend lives in a separate repo ([`vynno`](https://github.com/EmilM32/vynno)). It currently runs against mock `/mock/v1`. Pointing it at this API is frontend Phase 5c.

## Scope of this repository

**API only.**

| In this repo | Separate repo ([`vynno`](https://github.com/EmilM32/vynno)) |
| --- | --- |
| HTTP JSON API under `/v1` | SvelteKit UI, design system, i18n |
| Database and durable writes | HTTP-fetched mock JSON until swap |
| Authentication (Phase 3) | Stub login (`sessionStorage`) until swap |
| Server-side session and project rules | Client display, aggregates, theme, locale |

## Stack

| Layer | Choice |
| --- | --- |
| Language | Go |
| HTTP | Gin |
| Database | PostgreSQL (goose + sqlc, local Docker Compose) |
| Auth | None on the wire until Phase 3 ([ADR-0008](./docs/adr/0008-authentication.md) Deferred) |

Decisions: [ADR-0001](./docs/adr/0001-backend-stack.md), [ADR-0009](./docs/adr/0009-persistence.md).

## Run locally

Requires Go 1.26+ and Docker (for Postgres).

```sh
docker compose up -d
cp .env.example .env
set -a && source .env && set +a
go run ./cmd/api
```

`GET http://localhost:8080/healthz` → `{"status":"ok"}`.

`GET http://localhost:8080/v1/me` and the rest of [docs/api-contract.md](./docs/api-contract.md) are live. Point the SPA at `PUBLIC_API_BASE=http://localhost:8080/v1`. There is no auth yet. CORS is open (no credentials) until Phase 4.

```sh
go test ./...
golangci-lint run ./...
```

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

- **Phase 0:** Planning — done.
- **Phase 1:** Scaffold — done. See [docs/plans/phase-1-scaffold.md](./docs/plans/phase-1-scaffold.md).
- **Phase 2:** Contract v1 — in progress. See [docs/plans/phase-2-contract.md](./docs/plans/phase-2-contract.md).
