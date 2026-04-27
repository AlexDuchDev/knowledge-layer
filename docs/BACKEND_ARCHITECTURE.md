# Backend architecture

**Canonical:** [backend-architecture.md](backend-architecture.md), [apps:api.md](apps:api.md), [MODULE_BOUNDARIES.md](MODULE_BOUNDARIES.md).

**Module READMEs** under `apps/api/internal/modules/*/README.md` describe transport vs domain packages.

**Entrypoints:**

- API: `apps/api/cmd/api`
- Job worker: `apps/api/cmd/jobworker`
- Connector worker: `apps/api/cmd/connectorworker`

When package boundaries or startup behavior change, update [backend-architecture.md](backend-architecture.md) and relevant module READMEs.