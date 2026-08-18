# Plan — Phase 4 production (local host)

**Status:** Done  
**Last updated:** 2026-08-18  
**Tracking:** Roadmap Phase 4  
**Depends on:** Phase 3 Done, [ADR-0011](../adr/0011-local-production-host.md) Accepted

---

## Summary

Make this API something the owner can run every day on their machine: a release binary, Postgres in Docker, request logs, backups they have actually restored, and a short runbook. No cloud host.

## Why now

Phases 0–3 and profile/avatar are done. The SPA already talks to this API. The remaining P1 items are platform (PLAT-4 logs, PLAT-5 backups) plus a documented deploy path. Hosting is decided: local, single operator ([ADR-0011](../adr/0011-local-production-host.md)).

## Constraints

- No new `/v1` resources, fields, or error codes.
- `/healthz` and `/readyz` stay outside the SPA contract.
- Do not rebuild CORS; confirm loopback origins and `COOKIE_SECURE=false`.
- Do not `docker compose down -v` in start/stop scripts.
- `internal/domain` still imports neither Gin nor the database driver.
- Leave Phase 5 / backlog (pagination, prefs, LOG-6/7, insights, rate limit, OAuth) alone.

## Approach

1. Accept [ADR-0011](../adr/0011-local-production-host.md). Resolve open question #4. Point Phase 4 at this plan. Fix stale “Phase 3 current” / mock-handoff wording.
2. Structured request logs (`log/slog`). JSON when `LOG_FORMAT=json`. Do not log cookies or `Authorization`. Unexpected errors in `writeError` go to the logger; the client still gets the envelope.
3. `GET /readyz` pings Postgres (`200` / `503`). `/healthz` stays process-only.
4. Graceful shutdown on SIGINT / SIGTERM. `GIN_MODE=release` is set by the start script, not required for `go run` / tests.
5. `scripts/build`, `scripts/start` (`--detach` optional), `scripts/stop`, `scripts/backup`, `scripts/restore`. Compose starts Postgres only. API binary lives at `bin/vynno-api`.
6. Backup is `pg_dump` via `docker compose exec`. Restore replaces the database from a dump file. Document the drill; run it once against a throwaway dump.
7. `.env.example` lists `LOG_FORMAT` and the local-prod checklist (strong `BOOTSTRAP_PASSWORD`, do not wipe the volume).

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| `down -v` in a helper | Wipes the only copy of history | Scripts only `stop`; README warns |
| Restore while API is up | Writes during apply | `scripts/restore` stops the API first |
| Logging secrets | Token in stdout / log file | Never log cookie or Authorization |
| Treating `/readyz` as contract | Frontend schema drift | Not under `/v1`; README only |
| Cloud later | Local assumptions ossify | ADR-0011 says amend; no Dockerfile required now |

## Out of scope

Cloud host, Dockerfile as the primary run path, TLS, secret manager, login rate limit, disabling register, frontend production deploy, Phase 5 contract extensions.

## Exit checklist

- [x] ADR-0011 Accepted; open question #4 resolved
- [x] Roadmap Phase 4 is current; Phase 3 / avatar / handoff wording is no longer stale
- [x] Request logs on stdout; 5xx and unexpected errors logged
- [x] `GET /readyz` reflects Postgres; `GET /healthz` stays liveness
- [x] `scripts/build`, `start`, `stop`, `backup`, `restore`
- [x] Restore drill documented and executed once (2026-08-18: dump then restore the same file; tables intact)
- [x] `.env.example` + root README / AGENTS.md list the local-prod commands
- [x] `go test ./...` green
- [x] Roadmap Phase 4 boxes; this plan **Done**

## Related

- [../roadmap.md](../roadmap.md)
- [../adr/0011-local-production-host.md](../adr/0011-local-production-host.md)
- [../prd.md](../prd.md) PLAT-4, PLAT-5
- [../frontend-handoff.md](../frontend-handoff.md)
