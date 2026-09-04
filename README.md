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
| Mail | SMTP via `Mailer` port ([ADR-0015](./docs/adr/0015-outbound-email.md) Accepted) |
| Host | Owner’s machine ([ADR-0011](./docs/adr/0011-local-production-host.md) Accepted) |

Decisions: [ADR-0001](./docs/adr/0001-backend-stack.md), [ADR-0009](./docs/adr/0009-persistence.md), [ADR-0011](./docs/adr/0011-local-production-host.md).

## Run locally

Requires Go 1.26+ and Docker (for Postgres). Full runbook: **[docs/local-production.md](./docs/local-production.md)**.

Production and playground can run at the same time. They do **not** share rows. Do not edit `.env` to switch modes — `scripts/start` vs `scripts/dev` remap bind, database, public origin, mail mode, and `COOKIE_SECURE`.

| | Production (daily) | Playground |
| --- | --- | --- |
| Start | `./scripts/start` | `./scripts/dev` |
| Stop | `./scripts/stop` | `./scripts/stop --dev` |
| Bind | `127.0.0.1:27182` (`ADDR`) | `127.0.0.1:8081` (`DEV_ADDR`) |
| Database | `vynno` | `vynno_dev` |
| Process | `bin/vynno-api` | `go run ./cmd/api` |

`scripts/stop --postgres` stops Compose as well (named volume kept). It cannot be combined with `--dev`: Postgres is shared. Do **not** run `docker compose down -v`.

### Production

```sh
cp .env.example .env
./scripts/build
./scripts/start              # foreground; does not rebuild
./scripts/start --detach     # pid in var/api.pid, logs in logs/api.log
./scripts/status             # /healthz + /readyz
./scripts/stop               # production API only; Postgres stays up
./scripts/stop --postgres    # production API + Compose stop (named volume kept)
./scripts/backup
./scripts/restore backups/vynno-YYYYMMDD-HHMMSS.sql
```

`scripts/start` requires `bin/vynno-api` (`scripts/build` first). It does not create users: register from the SPA (`https://vynno.local`). Database `vynno` is daily history.

`GET http://127.0.0.1:27182/healthz` → `{"status":"ok"}` (process). `GET /readyz` is `200` when Postgres answers.

Operator API docs (generated from the Gin routes): [http://vynno.local:27182/swagger/](http://vynno.local:27182/swagger/) — open it at `PUBLIC_API_ORIGIN`, not `127.0.0.1`, so login cookies work. Spec: `GET /openapi.json`.

`/v1` requires a session (`GET /v1/avatars/:id` is the public exception, plus login/register/password-reset). Production: register from the SPA (`POST /v1/auth/register/code` then `/auth/register` with the 6-digit code). The SPA lists `https://vynno.local` in `SPA_ORIGIN` and uses same-origin `/v1`. Set `PUBLIC_API_ORIGIN=http://vynno.local:27182` so `avatarUrl` is an absolute URL (the SPA rewrites it to `/v1/avatars/…` on HTTPS). Set `COOKIE_SECURE=true` for the daily binary; `scripts/dev` forces `false`. `ADDR=127.0.0.1:27182`. Local mail catcher: Mailpit at http://127.0.0.1:8025 (Compose, SMTP `:1025`). First register needs Mailpit or real SMTP. Needs `/etc/hosts` `127.0.0.1 vynno.local`.

### Playground

```sh
./scripts/dev                # go run on 127.0.0.1:8081 → vynno_dev; Swagger at http://127.0.0.1:8081/swagger/
./scripts/stop --dev         # playground API only; Postgres stays up
./scripts/reset              # wipe vynno_dev → alexdev@vynno.local + Identity
./scripts/seed               # wipe vynno_dev → 3 demo accounts with history
```

Foreground: Ctrl-C in the `scripts/dev` terminal stops it. If `:8081` is already in use (`bind: address already in use`), a previous `go run` is still up — `scripts/stop --dev`, then `scripts/dev` again. `scripts/stop` without `--dev` does not touch the playground.

`reset` and `seed` are destructive (same confirm / `--yes` as `restore`). They refuse database `vynno` and do not stop the production API.

Demo logins after `scripts/seed` (playground only):

| Email | Password | Notes |
| --- | --- | --- |
| `alexdev@vynno.local` | `BOOTSTRAP_PASSWORD` from `.env` | Power user, live session, ~10 weeks of logs |
| `maya@vynno.local` | `SEED_PASSWORD` (default `local-dev-password`) | Contractor, idle |
| `rio@vynno.local` | same as Maya | Short history |

```sh
go test ./...
golangci-lint run ./...
./scripts/setup            # git config core.hooksPath .githooks
```

`git push` then runs `go test ./...`. Skip with `git push --no-verify`.

## Documentation

Start here: **[docs/README.md](./docs/README.md)**

| Doc | Description |
| --- | --- |
| [docs/api-contract.md](./docs/api-contract.md) | Wire JSON the SPA already speaks |
| [docs/domain-model.md](./docs/domain-model.md) | Entities and server-enforced rules |
| [docs/local-production.md](./docs/local-production.md) | Daily driver runbook |
| [docs/adr/](./docs/adr/) | Architecture decisions |
| [docs/backlog.md](./docs/backlog.md) | Deferred work |
