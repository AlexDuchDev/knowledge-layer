# Permission system

**Canonical documentation:** [permission-system.md](permission-system.md) and [permission-flow.md](permission-flow.md).

**Access model (HTTP + session):** [ACCESS_MODEL.md](ACCESS_MODEL.md).

**Code:** `apps/api/internal/identity_access/`, `apps/api/internal/platform/permissions/resolver.go`.

## Summary

Evaluation flows through **domain grants**, **roles**, **entity ACL**, **sensitivity**, and **policies**. Retrieval and Ask must **never** fetch unauthorized data and filter later.

When evaluation rules change, update the canonical docs above and any affected [API_SURFACE_V1.md](API_SURFACE_V1.md) entries.
