# Vynno API Documentation

Planning and architecture docs for the **Vynno backend**. This repository implements the HTTP API, persistence, and authentication. The SvelteKit UI lives in a separate frontend repository ([`vynno`](https://github.com/EmilM32/vynno)).

Brand: say **VIN-oh**. Public name is Vynno ([ADR-0007](./adr/0007-product-name.md)).

## Contents

| Document | Purpose |
| --- | --- |
| [working-agreement.md](./working-agreement.md) | How we write PRD, ADRs, roadmaps, and plans |
| [prd.md](./prd.md) | Product requirements, goals, priorities, out of scope |
| [domain-model.md](./domain-model.md) | Entities, relationships, server-enforced rules |
| [api-contract.md](./api-contract.md) | REST + JSON DTO contract the SPA already speaks |
| [frontend-handoff.md](./frontend-handoff.md) | How the frontend swaps from `/mock/v1` to this API |
| [local-production.md](./local-production.md) | Daily driver on this machine (binary + Compose) |
| [roadmap.md](./roadmap.md) | Phased backend delivery plan |
| [backlog.md](./backlog.md) | Deferred work (do not pull into the current phase) |
| [open-questions.md](./open-questions.md) | Undecided items with defaults |
| [adr/](./adr/) | Architecture Decision Records |
| [plans/](./plans/) | Implementation plans for a phase or feature |

## Stack

| Layer | Choice | ADR |
| --- | --- | --- |
| Language / HTTP | Go + Gin | [ADR-0001](./adr/0001-backend-stack.md) (Accepted) |
| Persistence | PostgreSQL, goose, sqlc | [ADR-0009](./adr/0009-persistence.md) (Accepted) |
| Auth | HttpOnly session cookie + remember-me | [ADR-0008](./adr/0008-authentication.md) (Accepted) |
| Mail | SMTP via `Mailer` port; Mailpit locally | [ADR-0015](./adr/0015-outbound-email.md) (Accepted) |
| Avatars | BYTEA + public `/v1/avatars/:id` | [ADR-0010](./adr/0010-avatar-storage.md) (Accepted) |
| Host | Owner’s machine; binary + Compose Postgres | [ADR-0011](./adr/0011-local-production-host.md) (Accepted) |
| Operator docs | OpenAPI from Gin routes; Swagger UI at `/swagger/` | [ADR-0013](./adr/0013-openapi-swagger.md) (Accepted) |

Also decided:

- Separate repository from the frontend — [ADR-0002](./adr/0002-separate-repository.md)
- HTTP JSON contract (`/v1`) — [ADR-0003](./adr/0003-http-json-contract.md)
- Project and session rules — [ADR-0004](./adr/0004-project-lifecycle.md), [ADR-0005](./adr/0005-session-lifecycle.md)
- Single-user v1 — [ADR-0006](./adr/0006-single-user-tenancy.md)
- User-defined activity types — [ADR-0012](./adr/0012-activity-types.md)
- OpenAPI / Swagger UI — [ADR-0013](./adr/0013-openapi-swagger.md)
- Outbound email — [ADR-0015](./adr/0015-outbound-email.md)

## Status

Phase 0–4 and profile/avatar are done. Session edit/delete/manual entry (LOG-6 / LOG-7) and session list pagination (PAGE) are done. Email login identifier is done. Outbound mail / register confirmation / password reset is in progress ([plans/email.md](./plans/email.md)). Phase 5 is later (prefs, insights).

See [roadmap.md](./roadmap.md).
