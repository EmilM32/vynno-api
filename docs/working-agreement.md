# Working agreement — how we document Vynno API

**Status:** Accepted  
**Last updated:** 2026-08-14

This is the process the frontend repository already follows, written down so this backend can start the same way.

We write things down **before** we build them, then keep the docs true as we go. We do not keep a second process (issue tracker, RFC board, design-doc loop) unless the product later needs one.

---

## 1. What lives where

| Document | Owns | Does not own |
| --- | --- | --- |
| [prd.md](./prd.md) | Vision, goals, non-goals, functional priorities, defaults for open questions | Stack, wire field names, implementation steps |
| [domain-model.md](./domain-model.md) | Entities, lifecycles, invariants the server must enforce | SQL schema, DTO JSON (that is the contract) |
| [api-contract.md](./api-contract.md) | Paths, methods, DTOs, status codes, error codes | Auth scheme (until ADR-0008), new resources |
| [roadmap.md](./roadmap.md) | Phases, checklists, exit criteria. **One current phase.** | Per-task implementation detail |
| [backlog.md](./backlog.md) | Deferred work that must not leak into the current phase | Current-phase tasks |
| [open-questions.md](./open-questions.md) | Undecided items + the default we ship with until decided | Accepted ADRs |
| [adr/](./adr/) | Expensive or irreversible technical choices | Day-to-day implementation notes |
| [plans/](./plans/) | A phase or feature that needs more than a roadmap checkbox | Product requirements (those stay in the PRD) |

`docs/README.md` is the index. If a doc is not linked from there, it does not exist.

---

## 2. When to write what

| Situation | Write |
| --- | --- |
| New product goal, priority, or non-goal | Update the PRD |
| New entity, invariant, or lifecycle rule | Update the domain model; ADR if it is a choice with real alternatives |
| New or changed HTTP resource, field, or error code | Amend [api-contract.md](./api-contract.md) **first**. Once the frontend repo exists alongside this one, change its `src/lib/api/schemas` and `docs/api-contract.md` in the same change set. |
| Language, framework, database, hosting, auth mechanism | ADR (Proposed → Accepted) |
| Starting a phase, or a feature with risks and sequenced steps | A plan under `docs/plans/` (same job as the frontend’s `ssr-enablement.md` / `api-next.md`) |
| Work we are deliberately not doing now | [backlog.md](./backlog.md) |
| We do not know yet | [open-questions.md](./open-questions.md) with a **default** |

If you can implement it by following an existing Accepted ADR and the contract, you do not need a new ADR.

---

## 3. Status vocabulary

Use these words in document headers. Do not invent nearby synonyms.

| Status | Meaning |
| --- | --- |
| **Draft** | Written, not yet the working agreement for that topic |
| **Proposed** | A concrete recommendation waiting for an accept/reject |
| **Accepted** | We will follow this |
| **Amended** | Still accepted; an amendment section records what changed |
| **Superseded** | Replaced by a later ADR; keep the file |
| **Deferred** | Not now; the backlog or a plan says when to revisit |

Roadmap checkboxes: `[ ]` not started, `[x]` done. Only check a box when the exit criterion is actually met.

---

## 4. ADRs

Template: [adr/TEMPLATE.md](./adr/TEMPLATE.md).

1. Next unused number, four digits: `0010-short-kebab-title.md`.
2. Capture **context, decision, consequences, alternatives**. That is the whole format.
3. **Do not rewrite history.** If the decision changes, add an `## Amendment (YYYY-MM-DD)` section, or write a new ADR that supersedes this one and update the index.
4. Update [adr/README.md](./adr/README.md) in the same change.
5. Deciders: “Project owner” unless more people are actually deciding.

Numbering continues in this repository only. Frontend ADR-0010 is not backend ADR-0010. Inherited decisions cite the frontend ADR they came from.

---

## 5. Plans

Template: [plans/TEMPLATE.md](./plans/TEMPLATE.md).

A roadmap row is enough when the work is obvious. Write a plan when:

- the phase has risks, sequencing, or “do not do X yet”
- a feature spans several days or several ADRs
- we need an exit checklist more specific than the roadmap

Name files after the work (`phase-0-planning.md`, `auth-session-cookies.md`), not after the date. Link the plan from the roadmap row and from `plans/README.md`.

When the work ships, mark the plan **Done** (or **Deferred**) and leave it. Do not delete it.

---

## 6. Contract discipline

The SPA is already wired to [api-contract.md](./api-contract.md). The mock engine in the frontend is the reference implementation of the rules.

- **Do not invent** resources, fields, or codes to “complete” the API.
- **Do not silently diverge.** If the live API must differ, change the contract and the frontend schemas — not just the server.
- `message` in the error envelope is for logs. The client maps `code` to UI strings.
- Out-of-scope items stay out until a contract amendment.

Cross-repo rule, once both repositories exist: a wire-format change is a **paired change**. Backend contract + frontend schemas + frontend `docs/api-contract.md`.

---

## 7. Phase 0 rule (same as the frontend)

No application code until Phase 0 exits. Exit is: product and architecture clear enough to scaffold without re-litigating stack, persistence, or the contract.

Proposed ADRs may be **Accepted** or **explicitly deferred with a default** in [open-questions.md](./open-questions.md). Either is a decision. Leaving them Proposed is not.

---

## 8. Voice

Match the frontend docs:

- Short sentences. Tables for inventories. Checklists for phases.
- State the default when something is open.
- Historical names (DevTime) may remain in inherited context. New prose says **Vynno**.
- Comments and docs explain non-obvious constraints, not the fact that we wrote a file.

---

## Related

- [prd.md](./prd.md)
- [roadmap.md](./roadmap.md)
- [adr/README.md](./adr/README.md)
- [plans/README.md](./plans/README.md)
