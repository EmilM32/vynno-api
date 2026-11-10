# ADR-0016: Read-only MCP for one account

**Status:** Accepted  
**Date:** 2026-09-25  
**Deciders:** Project owner

## Context

An agent working in this repository needs the owner's focus history: projects, sessions, the live timer, and time totals. That data lives in Postgres ([0009](./0009-persistence.md)). The same database holds `users.password_hash`, `auth_tokens.token_hash`, and `email_challenges.code_hash`.

The HTTP API already isolates accounts by the session token ([0008](./0008-authentication.md)). A general SQL tool would ignore that boundary and could read every account plus the secret columns. Aggregates stay off `/v1` ([domain model](../domain-model.md)); the agent can total time without a new public route.

## Decision

1. **Local stdio MCP** in `cmd/mcp`, built with `github.com/modelcontextprotocol/go-sdk/mcp`. Grok launches it from `.grok/config.toml`. Logs go to stderr.
2. **Same token table as the API.** `VYNNO_MCP_TOKEN` is an opaque secret stored like a session (since the 2026-11-10 amendment with scope `mcp`, which `/v1` refuses). The server hashes it and loads `auth_tokens`. A missing, unknown, or expired token refuses the process, and every tool resolves the token again.
3. **One account per process.** Tool arguments have no email and no user id. Queries take the user id from the token and include `user_id` in the `WHERE` clause. `vynno_whoami` returns that account's email and display name.
4. **Issuing a token is local.** `go run ./cmd/mcp token` reads `VYNNO_MCP_EMAIL` and `VYNNO_MCP_PASSWORD` from the environment or `.env`, runs the same login check as `POST /v1/auth/login`, and prints a new session token once. The operator stores it as `VYNNO_MCP_TOKEN` in `.env` (gitignored). It is not an HTTP route and it is not in the JSON contract. Password reset deletes every token for the account, including this one.
5. **Read-only.** Tools list projects, list sessions, read the live session, and summarize a time window. They do not start, stop, patch, or delete. Summary totals are computed here; `/v1` does not grow an aggregate endpoint.
6. **Column allowlist.** The MCP queries select project, session, profile, and activity-type fields. They do not select password hashes, token hashes, one-time-code hashes, or avatar bytes.
7. **Database.** `DATABASE_URL`, normally from `.env`, which points at `vynno`. The process does not run migrations.

| Tool | Returns |
| --- | --- |
| `vynno_whoami` | Authenticated email and display name |
| `vynno_list_projects` | That account's projects |
| `vynno_list_sessions` | That account's sessions, newest first, limit 50 (max 200) |
| `vynno_get_active_session` | The live timer, or idle |
| `vynno_summarize_time` | Overlap totals for a window, grouped by project, activity, or UTC day |

## Consequences

### Positive

- An agent can answer questions about one person's history without SQL.
- A token for account A cannot read account B, and cannot read secrets.
- Logout of that specific token, expiry, and password reset all cut the MCP off.
- The SPA contract is unchanged.

### Negative / tradeoffs

- The operator has to issue a token and keep it in `.env`. The SPA cookie is HttpOnly, so the browser cannot hand it over.
- Each `token` run inserts another `auth_tokens` row, same as another login.
- The MCP process holds the raw token in memory for as long as Grok keeps it running.
- Switching accounts means a new token and a new process.

## Amendment (2026-11-10) — the MCP token is not an API session

Decision 2 made `VYNNO_MCP_TOKEN` a full session: the same value worked as `Authorization: Bearer` on `/v1`. The MCP only reads, but a copy of that token from `.env` could start and delete sessions, change the password, and move the account to another email.

1. **`auth_tokens.scope`** is `session` or `mcp` (migration 00015; existing rows are `session`).
2. **`go run ./cmd/mcp token` issues scope `mcp`.** The HTTP API accepts only `session` tokens and answers `401 unauthorized` to an `mcp` token, as a cookie or as Bearer, exactly as it does to an unknown token. It does not revoke it.
3. **The MCP accepts both scopes.** A token issued before this amendment is a `session` token and keeps working until it expires. Run `go run ./cmd/mcp token` again to replace it with an `mcp` one; signing that old token out is `POST /v1/auth/logout` with it as Bearer, or a password change.
4. Lifetime (30 days), hashing, and revocation are unchanged: logout of all sessions by password reset, and password or email change from another session, delete the MCP token too.

The read-only guarantee no longer rests on the MCP's tool list alone. No wire change: `scope` is not in any DTO.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Stock Postgres MCP (`server-postgres`) | Arbitrary SQL, including secret columns and other accounts' rows. |
| Optional `email` argument when several accounts exist | Any connected agent could read every history in the database. |
| Email and password on the long-running process | Leaves the password in the agent environment for the whole session. A session token can be revoked on its own. |
| New `/v1` token or aggregate routes | The bearer transport and login check already exist. Aggregates stay off the SPA contract. |
| MCP OAuth | A second login protocol for a process on the same machine. |

## Related

- [0006-single-user-tenancy.md](./0006-single-user-tenancy.md)
- [0008-authentication.md](./0008-authentication.md)
- [0009-persistence.md](./0009-persistence.md)
- [../domain-model.md](../domain-model.md)
