# ADR-0015: Outbound email

**Status:** Accepted  
**Date:** 2026-08-26  
**Deciders:** Project owner

## Context

Login and register use an email address ([0008-authentication.md](./0008-authentication.md)). Nothing is sent to that address. Register confirmation and password reset need the process to deliver a one-time code.

The API runs on the owner’s machine ([0011-local-production-host.md](./0011-local-production-host.md)). There is no cloud account, no secret manager, and no public MX required for v1. A transactional SaaS SDK would add a vendor and an extra credential for no audience.

`internal/domain` must not import an SMTP client. Tests must not talk to a real provider. Playground seed users are `@vynno.local` and cannot receive mail on the public internet.

## Decision

1. **SMTP is the transport.** Env: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_STARTTLS`. From: `MAIL_FROM` (required for `smtp`), optional `MAIL_FROM_NAME`. A small SMTP client library is allowed. Do not take a Resend / SES / Postmark SDK. Those providers may still be used later as an SMTP endpoint via env.
2. **`Mailer` is a port** in `internal/mail`. `Send(ctx, Message{To, Subject, Text})`. `internal/domain` does not import it. `internal/service` depends on the interface, same pattern as `store.Store`.
3. **`MAIL_MODE` is `smtp` \| `log` \| `discard`.** Daily driver uses `smtp`. `log` is playground-only and may print the body (including a one-time code). `discard` is tests. `smtp` must not log the body or the code. Missing `SMTP_HOST` or `MAIL_FROM` in `smtp` mode fails boot.
4. **Local catcher is Mailpit** in Docker Compose (SMTP `1025`, UI `8025`). `.env.example` points SMTP at it. The API process does not require Mailpit unless `SMTP_HOST` names it.
5. **Plain text bodies for v1.** English. No template product, no i18n of mail, no marketing HTML. A later multipart HTML body is an amendment, not a new vendor.
6. **One-time codes belong to auth, not to this ADR.** Code format, TTL, hashing, and routes are [0008-authentication.md](./0008-authentication.md) and [../api-contract.md](../api-contract.md). This ADR only decides how a message leaves the process.
7. **Operator create skips mail.** `scripts/seed` / `scripts/reset` insert accounts through `cmd/devdata`. They do not call `Mailer`.

## Consequences

### Positive

- Register confirmation and password reset share one send path.
- Tests inject `discard` or a recording double; CI has no inbox.
- Moving to a real mailbox is env-only (Gmail app password, Fastmail, a provider’s SMTP).
- Mailpit gives a local inbox for SPA register e2e without a public MX.

### Negative / tradeoffs

- First daily `POST /auth/register` needs a reachable SMTP (Mailpit or a real host). A down catcher blocks new accounts, not login of existing ones.
- `log` mode prints secrets by design. It is not a production setting.
- SPF/DKIM/bounce handling is the SMTP provider’s job. This process will not interpret DSNs.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Resend / Postmark / SES SDK | Extra vendor and API key on a loopback host. SMTP can still speak to those later. |
| stdlib `net/smtp` only | AUTH and STARTTLS are painful enough that a small client library is cheaper. |
| Magic link as the only proof | Needs `SPA_ORIGIN` in the body and a click. Rejected for v1; codes are the auth decision. |
| Log-only forever | Cannot prove a real mailbox. Fine as a mode, not as the only mode. |
| Process-level send queue / worker | One user, loopback, few messages. Send in the request. Retry is the client’s. |

## Related

- [0008-authentication.md](./0008-authentication.md)
- [0011-local-production-host.md](./0011-local-production-host.md)
- [../api-contract.md](../api-contract.md)
- [../domain-model.md](../domain-model.md)
- [../local-production.md](../local-production.md)
