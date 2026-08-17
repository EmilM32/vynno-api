# Open questions — Vynno API

**Last updated:** 2026-08-17

Undecided items and the **default we ship with** until an ADR or PRD update says otherwise. When a question is resolved, move the outcome into an ADR (or the PRD) and mark the row resolved — do not delete the history.

| # | Question | Default until decided | Resolves via |
| --- | --- | --- | --- |
| 1 | Language / HTTP framework | **Resolved:** Go + Gin. Hand-written validation, `go test`, `gofmt` + `golangci-lint`. | [ADR-0001](./adr/0001-backend-stack.md) (Accepted) |
| 2 | Database engine | **Resolved:** PostgreSQL, goose, sqlc, `pgx`, local Docker Compose. | [ADR-0009](./adr/0009-persistence.md) (Accepted) |
| 3 | Auth mechanism | No auth on the wire until Phase 3. Cookie vs bearer still open. | [ADR-0008](./adr/0008-authentication.md) (Deferred) |
| 4 | Hosting / deploy target | Local process first; production host in Phase 4 | New ADR or 0001 amendment |
| 5 | Restrict project `color` to the SPA palette? | **No** — accept any `#rrggbb` | Amend [ADR-0004](./adr/0004-project-lifecycle.md) if we tighten |
| 6 | Empty session note | Store `"Untitled session"` (match the mock) | Already in [ADR-0005](./adr/0005-session-lifecycle.md) |
| 7 | Insights on the server? | **No** — client aggregates from `GET /sessions` | Contract amendment |
| 8 | Timezone storage | **UTC ISO-8601**; display TZ is the client | [domain-model.md](./domain-model.md) |
| 9 | ID format | **Opaque strings**; do not require `proj-` / `sess-` prefixes | [api-contract.md](./api-contract.md) |
| 10 | Pagination | **None** — `limit` only | Contract amendment when needed |
| 11 | User-defined activity types? | **No** — fixed enum | Contract amendment |
| 12 | Must every session have a project? | **Yes** | [domain-model.md](./domain-model.md) |
| 13 | Auto-stop previous session on start? | **No** | [ADR-0005](./adr/0005-session-lifecycle.md) |
| 14 | Shared DTO package with the frontend? | **No** — dual docs until we extract one | New ADR |
| 15 | Seed data for a fresh database | One profile + at least one active project | Phase 2 plan |

Rows 6, 8, 9, 11–13 already have defaults that are Accepted behavior. They stay here so Phase 0 does not re-ask them.

---

## Related

- [prd.md](./prd.md) §12
- [working-agreement.md](./working-agreement.md)
- [roadmap.md](./roadmap.md)
