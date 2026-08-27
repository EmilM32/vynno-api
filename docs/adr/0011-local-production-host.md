# ADR-0011: Local production host

**Status:** Accepted  
**Date:** 2026-08-18  
**Deciders:** Project owner

## Context

[ADR-0001](./0001-backend-stack.md) left hosting open (VPS, Fly, Cloud Run, …). [ADR-0009](./0009-persistence.md) deferred the production host.

The product is still a single-operator tool ([ADR-0006](./0006-single-user-tenancy.md)). The owner runs the stack on their own machine: a built API binary and PostgreSQL in Docker. Cloud hosting is not ruled out later.

The SPA already talks to this API locally (frontend Phase 5c). Cookies and CORS assume `SPA_ORIGIN` is a real browser origin; loopback HTTP is the current setup.

## Decision

1. **v1 production is the owner’s machine**, not a public cloud. A later public host amends this ADR (or writes a new one that supersedes it).
2. **PostgreSQL stays in Docker Compose** — the same `postgres` service as local development. Durable data is the Compose named volume. Do not `docker compose down -v` on that machine.
3. **The API is a compiled host binary** (`bin/vynno-api` from `./cmd/api`). It is not containerized in this decision. `scripts/build` writes the binary. `scripts/start` starts Compose, waits for Postgres, and runs the process; it does not rebuild.
4. **Secrets stay in a gitignored `.env`.** There is no cloud secret manager until there is a cloud host. `.env.example` remains the documented shape, not production values.
5. **CORS and cookies stay as [ADR-0008](./0008-authentication.md).** Loopback HTTP means `COOKIE_SECURE=false`. The daily SPA origin is `http://vynno.local` (Caddy on `127.0.0.1:80` in the SPA repo). Vite and preview stay on localhost. The operator API origin is `http://vynno.local:27182`.
6. **Backups are `pg_dump` through Compose** (`scripts/backup`, `scripts/restore`). Avatars are BYTEA, so they are in the dump. A restore drill is part of shipping this decision.
7. **Observability is structured logs on stdout** (JSON when `LOG_FORMAT=json`). No third-party error service. Unexpected handler errors are logged; the client still sees the contract envelope.
8. **`GET /healthz` is process liveness** (no DB). **`GET /readyz` pings Postgres.** Both are implementation details, not SPA contract resources.
9. **`POST /v1/auth/register` stays public.** The API is not on the public internet. Revisit register and login rate limits before any internet-facing host.
10. **The frontend remains a separate process** in the `vynno` repo. This ADR does not deploy the SPA.

## Consequences

### Positive

- Phase 4 can finish without picking Fly / Cloud Run / a VPS.
- Daily use is one script plus a running SPA.
- `pg_dump` matches what a future cloud host would use.
- A later cloud ADR can add a Dockerfile and a secret manager without changing the wire format.

### Negative / tradeoffs

- Availability is “the laptop is on.”
- `.env` on disk is the secret store. Fine for one operator; not fine for a shared host.
- HTTP on loopback; no TLS termination in this repo.
- Detached start writes logs to a local file, not a log drain.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Fly / Cloud Run / a VPS now | Owner is the only user; extra account, TLS, and cookie-domain work for no audience. |
| Put the API in Docker too | Works, but “built binary + Postgres in Docker” is what was asked. A Dockerfile can wait for a cloud amendment. |
| SQLite file to drop Compose | Rejected by [ADR-0009](./0009-persistence.md). |
| Skip backups because it is “just local” | Disk and volume mistakes still lose history. |

## Amendment (2026-08-19)

1. **Build and start are separate.** `scripts/build` writes `bin/vynno-api`. `scripts/start` does not rebuild; a missing binary is an error. Same split as the SPA.
2. **One Compose Postgres, two databases** on the named volume: `vynno` (daily binary, backup, restore) and `vynno_dev` (`scripts/dev`, `scripts/seed`, `scripts/reset`). Seed/reset refuse any other database name. `docker compose down -v` still wipes both and stays forbidden.
3. **The API process does not bootstrap accounts.** A fresh `vynno` has schema only. The first daily user is `POST /v1/auth/register` from the SPA. `BOOTSTRAP_*` is playground-only (`cmd/devdata`).
4. **Loopback bind.** Production `ADDR` is `127.0.0.1:27182`. Dev is `127.0.0.1:8081` so the daily binary and `go run` can run together. Register stays public (clause 9); loopback is the exposure control, matching the SPA.
5. **`scripts/stop` stops the API only.** `--postgres` also `docker compose stop` (volume kept).

## Amendment (2026-08-27)

Daily browser origin is **`http://vynno.local`**. Operator API origin is **`http://vynno.local:27182`**. `/etc/hosts` maps `vynno.local` to `127.0.0.1` only (do not add `::1` while listeners are IPv4). The `:80` reverse proxy is owned by the SPA repo.

Production `ADDR` moved off `:8080` to **`127.0.0.1:27182`** so a typical development day (other APIs on 8080) does not steal the bind. Playground stays `:8081`. Decision clause 5 and amendment-2019 clause 4 are updated in place.

## Related

- [0001-backend-stack.md](./0001-backend-stack.md)
- [0008-authentication.md](./0008-authentication.md)
- [0009-persistence.md](./0009-persistence.md)
- [../local-production.md](../local-production.md)
