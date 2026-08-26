# Plan — Email login identifier

**Status:** Done  
**Last updated:** 2026-08-26  
**Tracking:** Contract amendment (auth identifier)  
**Depends on:** [ADR-0008](../adr/0008-authentication.md) Accepted

---

## Summary

Login and register use an email address, not a username. Register fields are `email` + `password` + optional `displayName`. Profile `handle` (`@alexdev`) is removed. `ProfileDto` carries `email`. Chrome shows the display name if set, otherwise the raw email (no `@` prefix).

No mail is sent. No verification, password reset, or change-email.

## Why now

Username (`^[a-z0-9_]{3,32}$`) is the login identifier and the source of handle. The product wants an email address at register and login, and a single display string (name or email).

## Constraints

- Amend the contract (both repos) before handlers. Do not accept `username` as an alias.
- `internal/domain` imports neither Gin nor the database driver.
- No new endpoints. Error code `username_in_use` becomes `email_in_use`.
- Existing non-email usernames migrate to `{old}@vynno.local`.

## Approach

1. Amend ADR-0008, contract, domain model, PRD ME-2, handoff.
2. goose `00006_email_login.sql`: rename `users.username` → `email`; drop `profiles.handle`.
3. Domain: `NormalizeEmail`; empty display name allowed; no handle helpers.
4. Service + Gin DTOs: `email` on register/login and `ProfileDto`.
5. Seed/reset accounts are emails (`alexdev@vynno.local`, …).
6. Frontend pairing: schemas, login/register UI, Settings/SideNav, e2e.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| API ships before SPA | Login/register `invalid_response` | Paired change |
| Existing local accounts | `alexdev` no longer logs in | Migration appends `@vynno.local` |
| Email longer than display-name cap | Cannot copy email into `displayName` | Store empty name; UI falls back to email |

## Out of scope

Sending mail, verification, magic link, password reset, changing email after register, editable handle.

## Exit checklist

- [x] ADR-0008 amended; contract + domain + PRD updated
- [x] Frontend contract + Valibot + login/register UI
- [x] goose `00006` + sqlc
- [x] Register/login use `email`; `409 email_in_use`
- [x] Empty display name stored as `""`; PATCH `""` clears
- [x] Seed/reset accounts are emails
- [x] `go test ./...` green

## Related

- [../adr/0008-authentication.md](../adr/0008-authentication.md)
- [../api-contract.md](../api-contract.md)
- [../domain-model.md](../domain-model.md)
- [../frontend-handoff.md](../frontend-handoff.md)
