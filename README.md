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
| Authentication | Stub login until frontend Phase 5c; this API uses an HttpOnly cookie |
| Server-side session and project rules | Client display, aggregates, theme, locale |

## Stack

| Layer | Choice |
| --- | --- |
| Language | Go |
| HTTP | Gin |
| Database | PostgreSQL (goose + sqlc, local Docker Compose) |
| Auth | HttpOnly session cookie ([ADR-0008](./docs/adr/0008-authentication.md) Accepted) |

Decisions: [ADR-0001](./docs/adr/0001-backend-stack.md), [ADR-0009](./docs/adr/0009-persistence.md).

## Run locally

Requires Go 1.26+ and Docker (for Postgres).

```sh
docker compose up -d
cp .env.example .env   # then set BOOTSTRAP_PASSWORD if you want something other than the local default
go run ./cmd/api       # loads .env from the working directory
```

`GET http://localhost:8080/healthz` → `{"status":"ok"}`.

`GET http://localhost:8080/healthz` is public. `/v1` requires a session after Phase 3 (`GET /v1/avatars/:id` is the public exception): `POST /v1/auth/login` with the bootstrap username/password from `.env`. Point the SPA at `PUBLIC_API_BASE=http://localhost:8080/v1` and list that SPA origin in `SPA_ORIGIN`. Set `PUBLIC_API_ORIGIN=http://localhost:8080` so `avatarUrl` is an absolute URL.

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
- **Phase 2:** Contract v1 — done. See [docs/plans/phase-2-contract.md](./docs/plans/phase-2-contract.md).
- **Phase 3:** Auth — done. See [docs/plans/phase-3-auth.md](./docs/plans/phase-3-auth.md).
