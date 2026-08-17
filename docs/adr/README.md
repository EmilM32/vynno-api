# Architecture Decision Records

Lightweight ADRs for the Vynno **API** repository.

Inherited decisions cite the frontend ADR they came from. Numbers in this folder are this repository’s sequence, not the frontend’s.

| ADR | Title | Status |
| --- | --- | --- |
| [0001](./0001-backend-stack.md) | Backend stack | Accepted |
| [0002](./0002-separate-repository.md) | Separate repository from the frontend | Accepted |
| [0003](./0003-http-json-contract.md) | HTTP JSON contract is the API | Accepted |
| [0004](./0004-project-lifecycle.md) | Project lifecycle (archive + optional hard delete) | Accepted |
| [0005](./0005-session-lifecycle.md) | Session lifecycle | Accepted |
| [0006](./0006-single-user-tenancy.md) | Single-user tenancy for v1 | Accepted |
| [0007](./0007-product-name.md) | Public product name is Vynno | Accepted |
| [0008](./0008-authentication.md) | Authentication | Accepted |
| [0009](./0009-persistence.md) | Persistence | Accepted |

## Format

Each ADR captures: context, decision, consequences, alternatives. New decisions get the next number (`0010-…`). Template: [TEMPLATE.md](./TEMPLATE.md). Process: [../working-agreement.md](../working-agreement.md).
