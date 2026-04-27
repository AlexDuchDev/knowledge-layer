# Contributing to Knowledge Layer

Thank you for helping improve the platform. This document is the **extended** contributor guide; the short entry point remains [CONTRIBUTING.md](../CONTRIBUTING.md) at the repository root.

## Principles

- **Access control and auditability** take precedence over convenience.
- Prefer **small, reviewable** changes with tests for policy-sensitive logic (permissions, retrieval scope, ingestion gates).
- Follow existing layout: `apps/api/internal/` (Go modular monolith), `apps/web/src/` (Next.js).

## Documentation maintenance (mandatory)

Code and docs move together. Before opening a PR or finishing an agent task:

1. Read [DOCS_MAINTENANCE_POLICY.md](DOCS_MAINTENANCE_POLICY.md).
2. Use [DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md) to find which docs apply to the files you touched.
3. Walk [DOCS_UPDATE_CHECKLIST.md](DOCS_UPDATE_CHECKLIST.md) for the categories that apply.
4. Include the **completion report** from [templates/TASK_AND_PR_DOC_IMPACT.md](templates/TASK_AND_PR_DOC_IMPACT.md) in the PR description (or agent summary).

Skipping docs requires an explicit **justification** (see policy). “TODO later” is not acceptable for material behavior changes.

## Development setup

See the root [README.md](../README.md) for prerequisites, `.env.example`, database, API, web, and workers.

## Backend (API)

- Working directory: `apps/api`.
- Run tests: `go test ./...` (integration tests may need env such as `E2E_DB=1` per project convention).
- Migrations live under `apps/api/internal/db/migrations/`; API applies them on startup.

## Frontend (web)

- Working directory: `apps/web`.
- `npm install` then `npm run dev` for local development.
- `npm run build` before merging UI changes (catch type and build errors).

## Pull requests

- Describe **intent**, **scope**, and **risk** (especially auth, ingestion, governance, retrieval).
- Link related issues or **ADRs** under [adr/](adr/) when behavior or architecture changes.
- Complete the **documentation impact** section (use the template linked above).

## Architecture decisions (ADRs)

For durable, cross-cutting decisions, add or update a file in [adr/](adr/) using the existing numbering and style.

## Security

Report security issues per [SECURITY.md](../SECURITY.md); do not open public issues for undisclosed vulnerabilities.

## Releases and open-source hygiene

- Versioning and tags: [RELEASING.md](RELEASING.md).
- Before a public push or release: [`scripts/repo-sanity-check.sh`](../scripts/repo-sanity-check.sh) from the monorepo root.

## License

By contributing, you agree your contributions are licensed under the same terms as the project ([LICENSE](../LICENSE)).
