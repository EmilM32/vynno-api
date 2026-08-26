# Plan — Outbound email, register confirmation, password reset

**Status:** Draft (slice 0 docs landed; slices 1–3 not started)  
**Last updated:** 2026-08-26  
**Tracking:** Backlog AUTH-EXT (verification + reset only)  
**Depends on:** [ADR-0008](../adr/0008-authentication.md) Accepted; [email-login.md](./email-login.md) Done; [ADR-0015](../adr/0015-outbound-email.md) Accepted

---

## Summary

The API can send mail over SMTP. Registration does not create an account until the user submits a one-time code from that mail. A forgotten password is a reset (code + new password), never a reminder of the current secret.

OAuth, magic links, 2FA, passwordless login, and changing email after register stay backlog.

The live process still one-shot-registers until slices 1–3 land. This plan is the working agreement for that work. Do not deploy the breaking `RegisterDto.code` change without the SPA pairing.

## Why now

Email is the login identifier ([email-login.md](./email-login.md)) but nothing proves the address. Anyone can register as anyone’s mailbox. There is no recovery if the password is lost. AUTH-EXT deferred this until cookie sessions existed; they do.

A mailer inside `Register` would be copied for reset. The platform (SMTP, Mailpit, test double) lands first; the two product flows share it.

## Constraints

- Amend the contract (both repos) before handlers that change the wire. Do not accept `username` as an alias.
- `internal/domain` imports neither Gin, the database driver, nor `internal/mail`.
- Account is not created until the register code is accepted. No `emailVerified` on `ProfileDto`.
- Seed / reset / existing users skip mail. Login of those accounts is unchanged.
- Never email a password. Never put the one-time code in JSON. Never log the code in `smtp` mode.
- Single-user v1 ([ADR-0006](../adr/0006-single-user-tenancy.md)). No roles or invites.

## Product defaults

| Topic | Default |
| --- | --- |
| Proof | 6-digit numeric code, 15 minute TTL, SHA-256 at rest |
| Register | `POST /auth/register/code` then `POST /auth/register` with `code`; session cookie on success (same as today) |
| Taken email on register send | `409 email_in_use` |
| Password forgot | Always `204` for a well-formed email; send only if the account exists |
| Password reset | `204`, no session cookie; all `auth_tokens` for that user deleted |
| Resend | Replaces the previous challenge for that email+purpose |
| Cooldown | 60 seconds between sends; 5 sends / hour; 5 guesses then the challenge is spent |
| Existing accounts | Stay usable; no re-verification |

## Approach

Four slices. Do not merge reset handlers in the same commit as the first SMTP client.

### Slice 0 — Docs (this change)

ADR-0015, ADR-0008 amendment, contract, domain, PRD, backlog, roadmap, handoff, env example. Plan file is this document.

### Slice 1 — Mailer (no new `/v1` routes)

1. Config: `MAIL_MODE` (`smtp` \| `log` \| `discard`), `SMTP_*`, `MAIL_FROM`, `MAIL_FROM_NAME`. `smtp` requires host and from at boot.
2. `internal/mail`: `Mailer` port; SMTP / log / discard; recording test double.
3. Wire on `service.Service` like `Store`. HTTP tests that do not send mail use discard.
4. Compose `mailpit` (SMTP `:1025`, UI `:8025`). `.env.example` points SMTP at it.
5. Tests of log/discard/recording. Do not hit a real provider in CI.
6. Register remains one-shot until slice 2.

### Slice 2 — Register confirmation (breaking, paired SPA)

1. goose `email_challenges` (`email`, `purpose`, `code_hash`, `expires_at`, `attempt_count`). Unique `(email, purpose)`. No FK to `users`.
2. Domain: generate 6-digit `crypto/rand` code; validate exactly 6 digits; TTL / cooldown / guess cap.
3. `RequestRegisterCode` + `Register` requires a valid unused `purpose=register` challenge, then today’s create (account, profile, default project, cookie).
4. `POST /v1/auth/register/code` public. OpenAPI via `route()`.
5. Tests: happy path via recording mailer; wrong/expired/missing code; taken email; cooldown; seed/reset still skip mail.
6. SPA: two-step register; Paraglide for `invalid_code` and `rate_limited`; Playwright reads the code from Mailpit’s HTTP API. No test-only API backdoor.

### Slice 3 — Password reset (paired SPA)

1. Same table, `purpose=password_reset`.
2. `RequestPasswordReset` always `204` when the email is well-formed. `ResetPassword` verifies, bcrypts, `DeleteTokensByUser`, no cookie.
3. `POST /v1/auth/password/forgot` and `/reset`. Tests cover enumeration, session revoke, old password fail.
4. SPA: forgot-password from login. Logged-in change-password is a later authenticated route.

### Slice 4 — Operator docs

Local-production runbook: first daily register needs Mailpit or real SMTP. AUTH-EXT shrinks. This plan **Done**.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| API ships `code` before SPA | SPA `invalid_body` / `invalid_response` | Paired frontend change. Do not deploy the break alone. |
| 6-digit brute force | Guess a reset code | TTL, guess cap, cooldown, hash at rest |
| Forgot-password enumeration | Timing or distinct errors | Always `204`; send only if the user exists |
| SMTP down on first daily register | Cannot create the production user | Mailpit default; `log` for playground; document smtp must be reachable |
| OTP in `logs/api.log` | Secret on disk | Forbidden in `smtp` mode |
| Existing users locked out | Forced re-verify | Login unchanged; they are already users |
| Seed `@vynno.local` vs real MX | Bounce on production SMTP | Seed is `vynno_dev` + Mailpit only |

## Out of scope

OAuth, passkeys, magic links, 2FA, change-email, logged-in change-password, emailing the current password, bounce/DKIM product work, HTML marketing templates, `emailVerified` on `ProfileDto`, transactional SaaS SDKs, internet-facing RATE middleware.

## Exit checklist

- [ ] ADR-0015 Accepted; ADR-0008 amended; contract + domain + PRD + backlog + roadmap + handoff updated in **both** repos
- [ ] Mailer port with smtp / log / discard; Mailpit in Compose; `.env.example` documents `MAIL_*` / `SMTP_*`
- [ ] `POST /v1/auth/register` creates an account only with a valid unused register code
- [ ] `POST /v1/auth/register/code` and password forgot/reset exist with the documented codes
- [ ] Reset does not leak account existence; reset revokes sessions; old password fails
- [ ] Seed/reset/login of existing users do not require mail
- [ ] SPA register is two-step; forgot-password works; Paraglide maps `invalid_code` and `rate_limited`
- [ ] `go test ./...` green; OpenAPI from `route()` includes the new public ops
- [ ] No code values in smtp-mode logs; no secret in JSON

## Related

- [../adr/0015-outbound-email.md](../adr/0015-outbound-email.md)
- [../adr/0008-authentication.md](../adr/0008-authentication.md)
- [../api-contract.md](../api-contract.md)
- [../domain-model.md](../domain-model.md)
- [../roadmap.md](../roadmap.md)
- [email-login.md](./email-login.md)
