# Vynno API Documentation

Planning and architecture docs for the **Vynno backend**. This repository implements the HTTP API, persistence, and authentication. The SvelteKit UI lives in a separate frontend repository ([`vynno`](https://github.com/EmilM32/vynno)).

Brand: say **VIN-oh**. Public name is Vynno ([ADR-0007](./adr/0007-product-name.md)).

## Contents

| Document | Purpose |
| --- | --- |
| [working-agreement.md](./working-agreement.md) | How we write PRD, ADRs, and plans |
| [prd.md](./prd.md) | Product requirements, goals, non-goals |
| [domain-model.md](./domain-model.md) | Entities, relationships, server-enforced rules |
| [api-contract.md](./api-contract.md) | REST + JSON DTO contract the SPA already speaks |
| [local-production.md](./local-production.md) | Daily driver on this machine (binary + Compose) |
| [roadmap.md](./roadmap.md) | What has shipped; what is later |
| [backlog.md](./backlog.md) | Deferred work |
| [adr/](./adr/) | Architecture Decision Records |
| [plans/](./plans/) | In-flight implementation plans (none currently) |

## Stack

| Layer | Choice | ADR |
| --- | --- | --- |
| Language / HTTP | Go + Gin | [ADR-0001](./adr/0001-backend-stack.md) |
| Persistence | PostgreSQL, goose, sqlc | [ADR-0009](./adr/0009-persistence.md) |
| Auth | HttpOnly session cookie + remember-me | [ADR-0008](./adr/0008-authentication.md) |
| Mail | SMTP via `Mailer` port; Mailpit locally | [ADR-0015](./adr/0015-outbound-email.md) |
| Avatars | BYTEA + public `/v1/avatars/:id` | [ADR-0010](./adr/0010-avatar-storage.md) |
| Host | Owner’s machine; binary + Compose Postgres | [ADR-0011](./adr/0011-local-production-host.md) |
| Operator docs | OpenAPI from Gin routes; Swagger UI at `/swagger/` | [ADR-0013](./adr/0013-openapi-swagger.md) |

Also decided: separate repo ([ADR-0002](./adr/0002-separate-repository.md)), HTTP JSON `/v1` ([ADR-0003](./adr/0003-http-json-contract.md)), project and session rules ([ADR-0004](./adr/0004-project-lifecycle.md), [ADR-0005](./adr/0005-session-lifecycle.md)), single-user v1 ([ADR-0006](./adr/0006-single-user-tenancy.md)), activity types ([ADR-0012](./adr/0012-activity-types.md)), session list cursor ([ADR-0014](./adr/0014-session-list-pagination.md)).

Later work: [backlog.md](./backlog.md) / [roadmap.md](./roadmap.md).
