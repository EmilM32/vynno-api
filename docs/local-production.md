# Local production runbook

Daily driver for the production API **on this machine**. No cloud host. Decision: [ADR-0011](./adr/0011-local-production-host.md). Pair with the SPA runbook in the `vynno` repo.

```
browser  →  http://localhost:3000  (vynno, adapter-node)
                └── /v1 BFF  ──►  http://127.0.0.1:8080  (this repo, bin/vynno-api → database vynno)
                                        └── Postgres (Docker Compose, port 5433)
```

Playground (`scripts/dev`, seed, reset) uses database `vynno_dev` on `127.0.0.1:8081`. It does not share rows with daily history.

| | Production | Playground |
| --- | --- | --- |
| Start | `scripts/start` | `scripts/dev` |
| Stop | `scripts/stop` | `scripts/stop --dev` |
| Bind | `127.0.0.1:8080` | `127.0.0.1:8081` |
| Database | `vynno` | `vynno_dev` |

Use **`http://localhost:3000`** in the browser and in `SPA_ORIGIN`. `localhost` and `127.0.0.1` are different origins and do not share the session cookie.

## Once (or after API source changes)

```sh
cp .env.example .env   # if you do not already have one
# ADDR=127.0.0.1:8080
# DATABASE_URL → database vynno
# DEV_DATABASE_URL → database vynno_dev
# SPA_ORIGIN includes http://localhost:3000
./scripts/build
```

The API does **not** create a user. First visit: open the SPA and use the register tab. After [plans/email.md](./plans/email.md) slice 2, that tab emails a 6-digit code (Mailpit at [http://127.0.0.1:8025](http://127.0.0.1:8025) when SMTP points at Compose, or a real inbox). Until then register is still one-shot.

`BOOTSTRAP_*` / `SEED_PASSWORD` are only for `scripts/reset` and `scripts/seed` against `vynno_dev`. They are not production credentials.

### One-time: split an existing single database

If this volume already has playground rows in `vynno` and you have not isolated them yet:

1. `./scripts/backup` then `./scripts/stop`.
2. Start Postgres (`./scripts/start` will create `vynno_dev` if missing — stop the API again after, or `docker compose up -d` plus the create-db helper).
3. Restore that dump into `vynno_dev`.
4. Drop and create empty `vynno` (same SQL `scripts/restore` uses, targeting `vynno` only).
5. `./scripts/build && ./scripts/start --detach` — migrates, no users.
6. Register from `http://localhost:3000`.

Do not add a production wipe script. Do not `docker compose down -v`.

## Every day

```sh
./scripts/start           # foreground; Ctrl-C stops the API (Postgres stays up)
./scripts/start --detach  # pid in var/api.pid, logs in logs/api.log
```

Then in the `vynno` repo: `./scripts/start` (or `--detach`). Open [http://localhost:3000](http://localhost:3000). Operator API docs: [http://localhost:8080/swagger/](http://localhost:8080/swagger/) (must be this origin, matching `PUBLIC_API_ORIGIN`).

```sh
./scripts/stop            # detached production API only; Postgres stays up
./scripts/stop --postgres # production API + Compose stop (named volume kept)
```

`scripts/start` fails if `.env` is missing, `bin/vynno-api` is missing (`scripts/build` first), the API is already running, or `DATABASE_URL` is not database `vynno`.

After pulling API changes, run `scripts/build` again. Start does not rebuild.

## Playground (not daily history)

```sh
./scripts/dev             # go run on 127.0.0.1:8081 → vynno_dev; Swagger at http://127.0.0.1:8081/swagger/
./scripts/stop --dev      # playground API only; Postgres stays up
./scripts/seed            # wipe vynno_dev, load alexdev@vynno.local / maya@vynno.local / rio@vynno.local
./scripts/reset           # wipe vynno_dev, alexdev@vynno.local + Identity only
```

Foreground: Ctrl-C in the `scripts/dev` terminal stops it. A leftover `go run` child (closed terminal, `bind: address already in use` on `:8081`) is `scripts/stop --dev`. `scripts/stop` without `--dev` only stops the production binary in `var/api.pid`. `--dev` and `--postgres` cannot be combined.

Seed and reset refuse `vynno`. They do not stop the production API. Stop the playground first (`scripts/stop --dev`) before a wipe if `scripts/dev` is up.

Playwright in the SPA repo talks to `API_ORIGIN` (`:8080`) unless you set `E2E_API_BASE=http://localhost:8081/v1`. Use that override while the daily binary is on `:8080`, or e2e registers throwaway users into production.

## Backups

```sh
./scripts/backup
./scripts/restore backups/vynno-YYYYMMDD-HHMMSS.sql
```

Dump and restore are database `vynno` only. Avatars are BYTEA, so they are in the dump. Restore stops the API first.

## If login fails

1. Browser URL is `http://localhost:3000`, not `http://127.0.0.1:3000`.
2. This repo `SPA_ORIGIN` lists that exact origin (restart the API after editing `.env`).
3. `COOKIE_SECURE=false` (loopback HTTP).
4. Production has no `alexdev@vynno.local` unless you registered that address. Seed users live on `vynno_dev`. After the email-login migration, a leftover username `alexdev` logs in as `alexdev@vynno.local`.

## What this does not do

- Does not start the SPA. That is the `vynno` repo.
- Does not listen on the LAN (`ADDR=127.0.0.1:8080`).
- Does not rebuild on start.
- Does not create accounts. Register is the SPA.
- Does not `docker compose down -v`.
