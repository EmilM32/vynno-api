# Vynno API

**VIN-oh** — _Where the hours went._

This repository is the **backend** for Vynno: persistence, auth, and the HTTP JSON API the SvelteKit app already speaks.

The frontend lives in a separate repo ([`vynno`](https://github.com/EmilM32/vynno)). It talks to this API (`PUBLIC_API_BASE=/v1` locally). Mock `/mock/v1` is gone.

## Scope of this repository

**API only.**

| In this repo | Separate repo ([`vynno`](https://github.com/EmilM32/vynno)) |
| --- | --- |
| HTTP JSON API under `/v1` | SvelteKit UI, design system, i18n |
| Database and durable writes | HTTP client, theme, locale |
| Authentication | Login form; HttpOnly cookie `vynno_session` |
| Server-side session and project rules | Client display, aggregates, theme, locale |

## Stack

| Layer | Choice |
| --- | --- |
| Language | Go |
| HTTP | Gin |
| Database | PostgreSQL (goose + sqlc, local Docker Compose) |
| Auth | HttpOnly session cookie ([ADR-0008](./docs/adr/0008-authentication.md) Accepted) |
| Host | Owner’s machine ([ADR-0011](./docs/adr/0011-local-production-host.md) Accepted) |

Decisions: [ADR-0001](./docs/adr/0001-backend-stack.md), [ADR-0009](./docs/adr/0009-persistence.md), [ADR-0011](./docs/adr/0011-local-production-host.md).

## Run locally

Requires Go 1.26+ and Docker (for Postgres). Daily driver: **[docs/local-production.md](./docs/local-production.md)**.

```sh
cp .env.example .env
./scripts/build
./scripts/start              # foreground; does not rebuild
./scripts/start --detach     # pid in var/api.pid, logs in logs/api.log
./scripts/stop               # API only; Postgres stays up
./scripts/stop --postgres    # API + Compose stop (named volume kept)
./scripts/backup
./scripts/restore backups/vynno-YYYYMMDD-HHMMSS.sql
```

`scripts/start` requires `bin/vynno-api` (`scripts/build` first). It does not create users: register from the SPA (`http://localhost:3000`). Database `vynno` is daily history. Do **not** run `docker compose down -v`.

Playground (database `vynno_dev` only):

```sh
./scripts/dev                # go run on 127.0.0.1:8081
./scripts/reset              # wipe vynno_dev → alexdev + Identity
./scripts/seed               # wipe vynno_dev → 3 demo accounts with history
```

`reset` and `seed` are destructive (same confirm / `--yes` as `restore`). They refuse database `vynno` and do not stop the production API.

Demo logins after `scripts/seed` (playground only):

| Username | Password | Notes |
| --- | --- | --- |
| `alexdev` | `BOOTSTRAP_PASSWORD` from `.env` | Power user, live session, ~10 weeks of logs |
| `maya` | `SEED_PASSWORD` (default `local-dev-password`) | Contractor, idle |
| `rio` | same as Maya | Short history |

`GET http://127.0.0.1:8080/healthz` → `{"status":"ok"}` (process). `GET /readyz` is `200` when Postgres answers.

`/v1` requires a session (`GET /v1/avatars/:id` is the public exception). Production: `POST /v1/auth/register` then login. The SPA lists this origin in `SPA_ORIGIN` and uses same-origin `/v1`. Set `PUBLIC_API_ORIGIN=http://localhost:8080` so `avatarUrl` is an absolute URL. Leave `COOKIE_SECURE=false` on loopback HTTP. `ADDR=127.0.0.1:8080`.

```sh
go test ./...
golangci-lint run ./...
./scripts/setup            # git config core.hooksPath .githooks
```

`git push` then runs `go test ./...`. Skip with `git push --no-verify`.

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
- **Phase 4:** Local production — done. See [docs/plans/phase-4-production.md](./docs/plans/phase-4-production.md).
