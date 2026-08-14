# ADR-0002: Separate repository from the frontend

**Status:** Accepted  
**Date:** 2026-08-14  
**Deciders:** Project owner  
**Inherited from:** Frontend ADR-0002 (2026-08-12), “Frontend-only repository boundary”

## Context

Vynno needs a UI and, eventually, persistence, auth, and multi-device access. The frontend repository (`dev-time`) deliberately contains only the SvelteKit app. Mock `+server.ts` routes there are disposable and are not the system of record.

Coupling API, database, and UI in one repo now would have slowed the frontend and forced a stack choice before the contract existed. The contract now exists. The API still belongs in its own repository so stack, CI, and secrets stay independent.

## Decision

1. **This repository** contains only the Vynno **API** (HTTP handlers, domain rules, persistence, later auth).
2. **UI, design system, and i18n** stay in the frontend repository.
3. The shared language is [../api-contract.md](../api-contract.md), not a shared package (until both sides explicitly extract one).
4. No SvelteKit routes, Stitch assets, or frontend build tooling are introduced here.
5. After a live API exists, the frontend deletes `/mock/v1`. This repo never becomes a BFF for the SPA.

## Consequences

### Positive

- Backend can choose stack freely ([ADR-0001](./0001-backend-stack.md)) without rewriting views.
- Clear ownership and CI boundaries per repo.
- Secrets and migrations never land in the SPA repo.

### Negative / tradeoffs

- Dual maintenance of the contract until a shared spec is extracted.
- Cross-repo changes (new field, new error code) need a paired update ([../working-agreement.md](../working-agreement.md) §6).
- Local end-to-end requires running two processes.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Full-stack in the frontend repo | Rejected when the UI was built; still rejected. |
| Monorepo now | Extra packaging work; owner asked for a separate repo. |
| BaaS from day one (Supabase / Firebase) | Couples product to a vendor before stack is chosen; client SDK would bypass the existing contract. |

## Related

- [0003-http-json-contract.md](./0003-http-json-contract.md)
- [../frontend-handoff.md](../frontend-handoff.md)
- [../prd.md](../prd.md)
