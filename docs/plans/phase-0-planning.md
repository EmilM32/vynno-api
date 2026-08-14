# Plan — Phase 0 planning

**Status:** In progress  
**Last updated:** 2026-08-14  
**Tracking:** Roadmap Phase 0  
**Depends on:** nothing — this is the first plan

---

## Summary

Stand up the documentation habit for the Vynno API **before any application code**: PRD, domain model, contract snapshot, inherited ADRs, Proposed ADRs for stack / auth / persistence, and this process.

This kit was written in the frontend repository under `backend-docs/` so it can be copied into a new repo.

## Why now

The SPA is on Phase 5b (mock HTTP reads and writes). Phase 5c is a live origin. Starting the API repo as a blank folder would re-litigate decisions the frontend already made (one live session, archive rules, error envelope). Writing them down first is how the frontend itself started.

## Constraints

- Docs only. No scaffold.
- Do not invent endpoints or pick a stack in this phase.
- After copy, the new repo must be self-contained (no relative links into the frontend repo (`vynno`)).

## Approach

1. Copy `backend-docs/` (README, AGENTS.md, `docs/`) to the new repository root.
2. Trim the “How to use this folder” section from the new README.
3. Read [../working-agreement.md](../working-agreement.md), [../prd.md](../prd.md), [../api-contract.md](../api-contract.md).
4. Accept [ADR-0001](../adr/0001-backend-stack.md) (language + framework).
5. Accept [ADR-0009](../adr/0009-persistence.md) (engine + migrations).
6. Either accept [ADR-0008](../adr/0008-authentication.md) or write an amendment / open-questions note: “no auth on the wire until Phase 3.”
7. Check the remaining Phase 0 boxes on [../roadmap.md](../roadmap.md).
8. Only then start Phase 1 scaffold.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Scaffold before stack ADR | Premature framework lock-in | Phase 0 exit forbids code |
| Inventing “helpful” endpoints | SPA cannot call them; dual APIs | Contract discipline in AGENTS.md |
| Drift from frontend schemas | `invalid_response` in the SPA | Snapshot date + paired-change rule |
| Copying the kit but leaving it in `vynno` only | Two sources of truth | After copy, point the frontend `docs/README.md` at the new repo |

## Out of scope

- Choosing winners for the Proposed ADRs inside this plan (that is a product-owner decision).
- OpenAPI, CI, deploy, seed data beyond noting they belong later.
- Changing the frontend app.

## Exit checklist

- [x] Kit exists and is linked from the frontend docs index
- [x] PRD, domain, contract, roadmap, backlog, open questions, handoff
- [x] Working agreement + ADR and plan templates
- [x] Inherited ADRs 0002–0007 Accepted
- [x] Proposed ADRs 0001, 0008, 0009 written (not pre-decided)
- [ ] Kit copied to a new git repository
- [ ] ADR-0001 Accepted
- [ ] ADR-0009 Accepted
- [ ] ADR-0008 Accepted or explicitly deferred to Phase 3
- [ ] Roadmap Phase 0 boxes complete; this plan marked **Done**

## Related

- [../roadmap.md](../roadmap.md)
- [../working-agreement.md](../working-agreement.md)
- [../adr/README.md](../adr/README.md)
