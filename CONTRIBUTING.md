# Contributing

Thanks for helping improve Knowledge Layer. This guide gets you from a clean fork to a merged PR.

## TL;DR

```bash
# 1. Fork on GitHub, then locally:
git clone https://github.com/<your-fork>/knowledge-layer.git
cd knowledge-layer

# 2. Bring up infra + run the API
cp .env.example .env
make db-up
cd apps/api && go run ./cmd/api      # in one terminal

# 3. Bring up the web app
cd apps/web && npm install && npm run dev   # in another terminal

# 4. Branch + change + verify
git checkout -b your-feature
make lint typecheck test                     # all three should pass

# 5. Open a PR — fill in the doc-impact section in the PR template
```

## Before you start

Skim, in this order:

1. [README.md](README.md) — what the product is and is not.
2. [docs/OSS_V1_SCOPE.md](docs/OSS_V1_SCOPE.md) — the release contract: what ships, what's optional, what's deferred.
3. [docs/LIMITATIONS.md](docs/LIMITATIONS.md) — known stubs and intentional gaps.
4. [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) — extended guide (ADR process, security workflow, doc maintenance).
5. [AGENTS.md](AGENTS.md) — how AI agents (and humans) should approach changes here.

For the codebase map, read [`apps/api/internal/README.md`](apps/api/internal/README.md) — it lists every package and where to add new code.

## Local development

| Component | Command | Notes |
|---|---|---|
| Postgres + Redis + OpenSearch | `make db-up` | Docker Compose at the repo root. |
| Backend API | `cd apps/api && go run ./cmd/api` | Applies migrations on startup. |
| Job worker | `cd apps/api && go run ./cmd/jobworker` | Required when working on knowledge jobs or Second Brain. |
| Connector worker | `cd apps/api && go run ./cmd/connectorworker` | Required when ingesting through any connector. |
| Web app | `cd apps/web && npm install && npm run dev` | Next dev server on `:3000`. |

Smoke after the API is up:

```bash
bash scripts/smoke-local.sh   # header-auth, no UI
```

## Verification (before pushing)

These three are also CI gates:

```bash
make lint        # go vet, web ESLint (--max-warnings 0), shared TS check
make typecheck   # web + shared tsc --noEmit
make test        # go test ./... + builds worker binaries
```

For frontend changes, also run `npm run build` in `apps/web` — Phase 2 alignment locked the production build green.

**Optional but recommended for backend changes**: install [`golangci-lint`](https://golangci-lint.run/) and run

```bash
cd apps/api && golangci-lint run ./...
```

The repo ships [`apps/api/.golangci.yml`](apps/api/.golangci.yml) with a depguard rule that blocks unsanctioned imports of `internal/llm` (LLM clients must route through `ai/privacy.PrivacyGateway`) plus opinionated checks (errcheck, ineffassign, misspell, unused). It is not yet a CI gate, but PRs that add new violations slow down review.

## Pull requests

- **Branch off `main`**, one logical change per PR. If you find unrelated drift, file a follow-up issue rather than mixing it in.
- **Fill in the [PR template](.github/pull_request_template.md) completely**, including the documentation-impact section. Per [docs/DOCS_MAINTENANCE_POLICY.md](docs/DOCS_MAINTENANCE_POLICY.md), code that touches a documented area must update the doc in the same PR — see [docs/DOCS_IMPACT_MAP.md](docs/DOCS_IMPACT_MAP.md) for the mapping.
- **Tests**: any policy-sensitive logic (access checks, sanitization, audit emission, dedup) needs a test. Adapter changes need at least one unit test (Slack adapter is the reference for OSS-quality coverage).
- **ADRs**: behavioural changes that constrain future work go under `docs/adr/`. Number monotonically; copy ADR-0013 as a template.
- **Pre-push hygiene**: run `scripts/repo-sanity-check.sh` once before your first push to confirm no `.env` / `.pem` is tracked. The script also runs in CI.

The PR description should answer: *what changes, why, and what could break*. Risk lines about auth, ingestion, or governance changes are read closely.

## Filing issues

- **Bug** → `.github/ISSUE_TEMPLATE/bug_report.md`. Include reproduction, environment, and `/audit-events` / `/ops/failed-runs` checks already done.
- **Feature** → `.github/ISSUE_TEMPLATE/feature_request.md`. Walk through the governance checklist (access / audit / privacy / provenance / failure mode) — this is how every feature gets prioritised.
- **Docs** → `.github/ISSUE_TEMPLATE/docs_improvement.md`. PRs are welcome and usually faster than an issue.
- **Security** → **do not open a public issue.** See [SECURITY.md](SECURITY.md) for the coordinated disclosure path.

## What we look for

- **Access control and auditability take precedence over convenience.** A more elegant flow that bypasses `AccessEvaluator` will not merge.
- **Small, reviewable changes** with explicit risk callouts. Two PRs ship faster than one large one.
- **Existing patterns** before new abstractions. Read the package next door before introducing one of your own.
- **Honest LIMITATIONS.md updates** when behaviour crosses from stub to supported (or back).

## Optional modules

Some surfaces (Second Brain, GraphRAG) are feature-flag-gated optional modules per [docs/OSS_V1_SCOPE.md](docs/OSS_V1_SCOPE.md). PRs that add new optional modules must:

1. Document the enabling env flag(s) in `.env.example` and [docs/CONFIG_ENV.md](docs/CONFIG_ENV.md).
2. Default to disabled — a fork that ignores the flag must still pass `make test` and `bash scripts/smoke-local.sh`.
3. Add an entry to `docs/OSS_V1_SCOPE.md` "Optional modules" section.

## Releases

The project uses semver from v1.0; v0.x allows breaking changes between minor versions (see [docs/API_STABILITY.md](docs/API_STABILITY.md)). Release tagging conventions live in [docs/RELEASING.md](docs/RELEASING.md).

## License

By contributing, you agree that your contributions are licensed under the same terms as the project ([LICENSE](LICENSE) — Apache 2.0).
