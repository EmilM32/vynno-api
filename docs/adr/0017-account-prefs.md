# ADR-0017: Account preferences

**Status:** Accepted  
**Date:** 2026-09-30  
**Deciders:** Project owner

## Context

The SPA kept the daily target and the default project in a device cookie (`vynno_prefs`) so SSR and hydrate shared one snapshot (frontend ADR-0011). Setting a target on the laptop did nothing on the desktop. The contract listed prefs as out of scope and the backlog tracked them as PREFS.

Theme and locale are also device-local. They are applied before first paint (theme from `localStorage` in `app.html`, locale from a cookie), and a person may want a light theme on one machine and dark on another.

## Decision

1. **New resource `GET` / `PATCH /v1/me/prefs`.** Body `PrefsDto { dailyTargetMs, defaultProjectId }`. Unset values are JSON `null`; the SPA keeps its own defaults (8 hours, first active project).
2. **Account-wide, not per device.** One row per user in `user_prefs`. No row means every pref is unset, so existing accounts need no backfill.
3. **PATCH follows the present-vs-absent rule.** Omit leaves a field unchanged; `null` clears it. Unknown fields are `400 invalid_body`.
4. **`dailyTargetMs`** is an integer from 60000 through 86400000 (one minute to one day). Anything else is `400 invalid_body`.
5. **`defaultProjectId`** must be a project this user owns. Unknown, malformed, or another user's id is `404 not_found`, the same as `activityTypeId` on sessions. Archived is allowed: archiving a project does not rewrite prefs, and the SPA falls back to an active project.
6. **Hard-deleting the default project clears it** (`ON DELETE SET NULL`) instead of blocking the delete.
7. **Theme and locale stay device-local.** They are not fields on this resource.
8. **The resource is extensible by amendment only.** A new pref is a contract change, not a free-form key.

## Consequences

### Positive

- The daily target and default project follow the user to every device and browser.
- SSR still has the values on first paint: the SPA's layout load fetches them with the rest of the seed.
- No migration for existing users; the SPA can copy an old `vynno_prefs` cookie up once.

### Negative / tradeoffs

- One more request in the SPA seed.
- Two devices editing the same pref: last write wins. There is no version field.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Fields on `ProfileDto` / `PATCH /me` | Profile is identity (name, email, avatar). Mixing settings in makes every profile write carry them, and the avatar endpoints return `ProfileDto`. |
| Free-form key-value blob | The server could not validate values or project ownership, and the contract would stop describing the wire. |
| Keep the device cookie | The reason for this change: settings did not follow the user. |
| Include theme and locale | They must apply before first paint on each device, and per-device choice is a feature. |

## Related

- [../api-contract.md](../api-contract.md) Preferences
- [../domain-model.md](../domain-model.md)
- [0006-single-user-tenancy.md](./0006-single-user-tenancy.md)
