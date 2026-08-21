# Plan — Swagger / OpenAPI

**Status:** Done  
**Last updated:** 2026-08-21  
**Tracking:** Backlog `OPENAPI` (operator docs; not a SPA contract amendment)  
**Depends on:** [ADR-0013](../adr/0013-openapi-swagger.md) Accepted, [ADR-0008](../adr/0008-authentication.md)

---

## Summary

Interactive Swagger UI for every live HTTP route. The OpenAPI 3.0 document is **generated from the same Go `route()` calls that mount Gin handlers**, using the DTO structs those handlers already use. It is not a second contract and does not generate a frontend client.

- Production: `http://localhost:8080/swagger/`
- Playground: `http://127.0.0.1:8081/swagger/`
- Spec: `GET /openapi.json`

## Why now

Curl against the markdown contract is slow. A hand-written YAML would rot. Generating from the mount point keeps the UI aligned with the process that is actually listening.

## Constraints

- Do not add `/v1` resources, fields, query params, or error codes.
- `internal/domain` imports neither Gin nor Swagger libraries.
- Validation stays hand-written.
- `docs/api-contract.md` remains the SPA/human contract.

## Approach

1. Accept ADR-0013. Amend ADR-0008: CSRF allowlist includes `PUBLIC_API_ORIGIN`.
2. Wrap API route registration (`Server.route`) so a Gin path cannot exist without OpenAPI metadata.
3. Build OpenAPI 3.0 JSON at process start; schemas from DTO `json` tags.
4. Serve `/openapi.json` and embedded Swagger UI at `/swagger/` (`swgui` v5, `withCredentials`).
5. `scripts/dev` sets playground `PUBLIC_API_ORIGIN` to the bind origin so Try-it-out cookies work on `:8081`.
6. Test: Gin routes ↔ spec paths; CSRF allows the API origin; `/swagger/` is HTML.

## Risks

| Risk | Failure mode | Mitigation |
| --- | --- | --- |
| Incomplete `op` metadata | Path exists, errors/descriptions wrong | Inventory test for paths; contract tables when adding routes |
| localhost vs 127.0.0.1 | Cookie Try-it-out 401 | Open Swagger at `PUBLIC_API_ORIGIN` |
| Dual contract | Spec invents fields | Schemas from existing DTOs only |

## Out of scope

Generated clients, replacing `api-contract.md`, OpenAPI-first design, auth-gating `/swagger/`, Scalar/Redoc, pagination/prefs/insights.

## Exit checklist

- [x] ADR-0013 Accepted; ADR-0008 amended; this plan filed
- [x] Spec generated from `route()`; covers ops + `/v1`
- [x] `GET /swagger/` and `GET /openapi.json`
- [x] CSRF allows `PUBLIC_API_ORIGIN`; playground origin aligned
- [x] Route-inventory test
- [x] Operator docs mention the URL

## Related

- [../adr/0013-openapi-swagger.md](../adr/0013-openapi-swagger.md)
- [../api-contract.md](../api-contract.md)
- [../roadmap.md](../roadmap.md)
