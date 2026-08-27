# ADR-0010: Avatar storage

**Status:** Accepted  
**Date:** 2026-08-17  
**Deciders:** Project owner

## Context

`profiles.avatar_url` has existed since Phase 2 as a nullable TEXT column, and `ProfileDto.avatarUrl` is already on the wire. Nothing wrote either. Settings needs a real photo: pick a file, persist it, show it in chrome.

A client-supplied URL is hotlinking (SSRF if we fetch it, broken images if the remote dies). Base64 in `PATCH /me` inflates the JSON contract. v1 has no object-storage bucket ([ADR-0011](./0011-local-production-host.md)). One image per user, hard-capped at 1 MiB.

`<img src>` from the SPA origin to the API origin is a cross-site subresource. `SameSite=Lax` cookies are not sent on that request, so a cookie-gated `/me/avatar` would be a broken image.

## Decision

1. **Bytes live in PostgreSQL.** Table `avatars` (`id`, `user_id` UNIQUE, `content_type`, `bytes`, `created_at`). `BYTEA`. One live row per user.
2. **Pointer stays on `profiles.avatar_url`.** Store the public path `/v1/avatars/{uuid}`. JSON `null` when absent. `GET /me` never selects `bytes`.
3. **Serve at `GET /v1/avatars/:id` without a session.** Avatars are display photos, not secrets. The path id is a UUID so it is unguessable and cache-busts on replace.
4. **Wire `avatarUrl` is absolute.** `{PUBLIC_API_ORIGIN}` + stored path. Origin is process config; rows are not rewritten when the host changes.
5. **Writes are authenticated.** `PUT /v1/me/avatar` (multipart field `file`) and `DELETE /v1/me/avatar`. Magic-byte sniff (JPEG / PNG / WebP only). Max 1 MiB. No transcode in this decision.
6. **A `BlobStore` port may replace BYTEA later** (disk or S3) without a wire change. The contract publishes a URL, not a storage engine.

## Consequences

### Positive

- `pg_dump` already planned for Phase 4 includes the photo.
- No extra volume or bucket before a host is chosen.
- Profile reads stay thin.
- SPA `<img src={avatarUrl}>` works cross-origin without cookie gymnastics.

### Negative / tradeoffs

- `pg_dump` grows by up to 1 MiB per user. Acceptable for v1.
- Public GET means anyone with the UUID can load the image. Treat the URL as unguessable, not classified.
- Amends [ADR-0008](./0008-authentication.md) decision 6: one more public `/v1` GET.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Client-supplied `avatarUrl` | Not a photo upload. SSRF / hotlink. |
| Base64 in `PATCH /me` | Inflates JSON; still need a GET. |
| `BYTEA` on `profiles` | Every profile read would have to list columns or load megabytes. |
| Local filesystem | Extra backup surface before a host exists. |
| S3 / R2 / MinIO | Hosting ADR does not exist. Overkill for one thumbnail. |
| Cookie-auth image GET | Cross-origin `<img>` will not send `vynno_session`. |

## Related

- [../api-contract.md](../api-contract.md)
- [../domain-model.md](../domain-model.md)
- [0008-authentication.md](./0008-authentication.md)
- [0009-persistence.md](./0009-persistence.md)
