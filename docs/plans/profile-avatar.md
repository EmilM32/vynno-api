# Plan — Profile edit and avatar upload

**Status:** Done  
**Last updated:** 2026-08-17  
**Tracking:** ME-3 / ME-4 / ME-5, Roadmap Phase 5  
**Depends on:** Phase 3 Done, [ADR-0010](../adr/0010-avatar-storage.md) Accepted

---

## Summary

A signed-in user can change their display name and upload or remove a photo from Settings. Registration stays unchanged (`avatarUrl` starts null). Bytes live in PostgreSQL; the wire still publishes `ProfileDto.avatarUrl` as an absolute public URL.

## Why now

The column and DTO already reserved a photo. Settings still shows initials and mock copy. `PATCH /me` was backlog ME-3 with no upload path.

## Constraints

- Amend the contract (both repos) before handlers.
- No new error codes. Oversize / bad type / missing file are `invalid_body`.
- `internal/domain` imports neither Gin nor the database driver.
- Do not accept a client-supplied `avatarUrl` string.
- Handle stays derived from the username.

## Approach

1. Accept [ADR-0010](../adr/0010-avatar-storage.md); amend contract, domain, PRD, backlog, roadmap, handoff.
2. goose `00003_profile_avatar.sql`: `avatars` table. sqlc + Memory.
3. Domain: required display name; JPEG/PNG/WebP magic bytes; 1 MiB cap.
4. Service: patch, replace (new UUID), delete, compose `{PUBLIC_API_ORIGIN}` + stored path.
5. Gin: `PATCH /me`, `PUT /me/avatar`, `DELETE /me/avatar`, public `GET /avatars/:id`. CORS allows `PUT`.
6. Frontend pairing: schemas, `putFile`, Settings + SideNav.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Cookie-gated image URL | Broken `<img>` cross-origin | Public `GET /avatars/:id` |
| Absolute URL stored in DB | Origin change orphans `src` | Store path; prefix at DTO time |
| Trusting `Content-Type` | HTML/SVG as `image/jpeg` | Magic-byte sniff |
| Huge multipart | Memory spike | 1 MiB domain cap + Gin multipart memory cap |

## Out of scope

Avatar on register, editable handle, client-supplied URLs, transcode/crop, object storage, Gravatar, prefs persistence.

## Exit checklist

- [x] ADR-0010 Accepted; contract amended in this repo
- [x] Contract + Valibot amended in the frontend repo
- [x] `PATCH /me` updates `displayName`; handle untouched
- [x] `PUT /me/avatar` stores jpeg/png/webp ≤ 1 MiB and returns a loadable `avatarUrl`
- [x] `DELETE /me/avatar` clears to `null` and deletes bytes
- [x] `GET /v1/avatars/:id` works without a cookie
- [x] Register / login still create `avatarUrl: null`
- [x] Unauthenticated writes → `401 unauthorized`
- [x] Bad file → `400 invalid_body`
- [x] Settings can add, replace, and remove a photo; SideNav shows it
- [x] `go test ./...` green

## Related

- [../adr/0010-avatar-storage.md](../adr/0010-avatar-storage.md)
- [../api-contract.md](../api-contract.md)
- [../roadmap.md](../roadmap.md)
