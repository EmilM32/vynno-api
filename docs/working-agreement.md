# Working agreement — how we document Vynno API

**Status:** Accepted  
**Last updated:** 2026-08-27

We write things down **before** we build them, then keep the docs true as we go. We do not keep a second process (issue tracker, RFC board) unless the product later needs one.

Canonical sources: **contract, domain, ADRs, runbook**. Plans exist only while work is in flight.

---

## 1. What lives where

| Document | Owns | Does not own |
| --- | --- | --- |
| [prd.md](./prd.md) | Vision, goals, non-goals | Stack, wire field names, implementation steps |
| [domain-model.md](./domain-model.md) | Entities, lifecycles, invariants the server must enforce | SQL schema, DTO JSON (that is the contract) |
| [api-contract.md](./api-contract.md) | Paths, methods, DTOs, status codes, error codes | Auth *why* (that is ADR-0008) |
| [roadmap.md](./roadmap.md) | What has shipped; what is later | Per-task implementation detail |
| [backlog.md](./backlog.md) | Work we are deliberately not doing now | Current in-flight tasks |
| [local-production.md](./local-production.md) | How to run, mail, backup, two databases | Product requirements |
| [adr/](./adr/) | Expensive or irreversible technical choices | Day-to-day implementation notes |
| [plans/](./plans/) | A feature in flight that needs more than a roadmap row | Product requirements (those stay in the PRD) |

`docs/README.md` is the index. If a doc is not linked from there, it does not exist.

---

## 2. When to write what

| Situation | Write |
| --- | --- |
| New product goal, priority, or non-goal | Update the PRD |
| New entity, invariant, or lifecycle rule | Update the domain model; ADR if it is a choice with real alternatives |
| New or changed HTTP resource, field, or error code | Amend [api-contract.md](./api-contract.md) **first**. Change the frontend `src/lib/api/schemas` and `docs/api-contract.md` in the same change set. |
| Language, framework, database, hosting, auth mechanism | ADR (Proposed → Accepted) |
| A feature with risks, sequencing, or “do not do X yet” | A plan under `docs/plans/` |
| Work we are deliberately not doing now | [backlog.md](./backlog.md) |

If you can implement it by following an existing Accepted ADR and the contract, you do not need a new ADR or a plan.

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
| **Deferred** | Not now; the backlog says when to revisit |

---

## 4. ADRs

Template: [adr/TEMPLATE.md](./adr/TEMPLATE.md).

1. Next unused number, four digits: `0010-short-kebab-title.md`.
2. Capture **context, decision, consequences, alternatives**. That is the whole format.
3. **Do not rewrite history.** If the decision changes, add an `## Amendment (YYYY-MM-DD)` section, or write a new ADR that supersedes this one and update the index. The Decision section may be updated in place so a current reader sees the live rules; the amendment records what changed.
4. Update [adr/README.md](./adr/README.md) in the same change.
5. Deciders: “Project owner” unless more people are actually deciding.

Numbering continues in this repository only. Frontend ADR-0010 is not backend ADR-0010. Inherited decisions cite the frontend ADR they came from.

Do not delete ADRs.

---

## 5. Plans

Template: [plans/TEMPLATE.md](./plans/TEMPLATE.md).

A backlog or roadmap row is enough when the work is obvious. Write a plan when the feature spans several days, several ADRs, or needs an explicit “do not do X yet.”

Name files after the work (`auth-session-cookies.md`), not after the date. Link the plan from `plans/README.md` and from `docs/README.md` while it is in flight.

When the work ships, move unique facts into the contract, domain, ADR, and/or runbook, then **delete the plan**. Git history is the archive. Do not keep Done plans in the tree.

---

## 6. Contract discipline

The SPA is wired to [api-contract.md](./api-contract.md).

- **Do not invent** resources, fields, or codes to “complete” the API.
- **Do not silently diverge.** If the live API must differ, change the contract and the frontend schemas — not just the server.
- `message` in the error envelope is for logs. The client maps `code` to UI strings.
- Out-of-scope items stay out until a contract amendment.

A wire-format change is a **paired change**: backend contract + frontend schemas + frontend `docs/api-contract.md`.

---

## 7. Voice

- Short sentences. Tables for inventories. Checklists for in-flight work.
- State the default when something is open.
- Historical names (DevTime) may remain in inherited context. New prose says **Vynno**.
- Comments and docs explain non-obvious constraints, not the fact that we wrote a file.
