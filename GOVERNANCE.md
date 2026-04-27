# Project governance

How decisions get made on Knowledge Layer. Short, practical, and aligned with how the project actually operates today (small contributor base, single-tenant scope, Apache 2.0).

## Roles

| Role | Who | What they can do |
|---|---|---|
| **Maintainer** | Anyone with merge / release authority on the canonical repo. | Approve and merge PRs, cut releases, accept ADRs, manage issue / discussion moderation. |
| **Contributor** | Anyone who opens an issue, PR, or discussion. | Propose changes, file ADRs, request reviews. |
| **Operator** | Anyone running a self-hosted instance. | Open feature requests, file bugs, share deployment feedback. They are the primary audience for design tradeoffs. |

There is no formal core-team distinction beyond "maintainer." The list of current maintainers is whoever has commit access on the canonical repo's default branch — visible via `git log` history and the GitHub *Insights → People* page.

## How changes get in

1. **Discussion first for behavioural changes.** If a PR will change an externally visible API contract, an access-control rule, an audit event, or a default behaviour an operator depends on, open an issue or draft ADR before writing the code. This avoids re-work and gives operators a chance to weigh in.
2. **One logical change per PR.** Per [`CONTRIBUTING.md`](CONTRIBUTING.md). Drive-by refactors get filed as follow-up issues, not bundled in.
3. **Documentation must stay aligned.** Per [`docs/DOCS_MAINTENANCE_POLICY.md`](docs/DOCS_MAINTENANCE_POLICY.md), code that touches a documented area must update the doc in the same PR. The PR template's *Documentation impact* section is read closely.
4. **Tests for policy-sensitive logic.** Access checks, sanitization, audit emission, dedup — all need tests. The Slack and Mattermost webhook adapters are reference quality.
5. **Maintainer review and merge.** At least one maintainer reviews. Trivial doc / typo PRs may self-merge if the author is a maintainer; behavioural PRs always need a second pair of eyes.

If a PR sits without review for **more than 7 calendar days**, the contributor may ping `@maintainers` (or the relevant area owner if known) on the PR. Maintainers aim to triage every PR within that window even when full review takes longer.

## Architectural decisions (ADRs)

Behavioural changes that constrain future work go under [`docs/adr/`](docs/adr/). The pattern:

1. Copy the most recent ADR (currently [ADR-0014](docs/adr/0014-single-tenant-deployment-stance.md)) as a template.
2. Number monotonically; never reuse a number.
3. Status starts at **Provisional**. It promotes to **Accepted** only after a maintainer merges the ADR and at least one release cycle has passed without revisit triggers firing.
4. Each ADR documents a **revisit gate** — the conditions under which a future ADR could supersede it.

ADR-0014 is the reference for the revisit-gate pattern: it pre-commits to the schema-migration plan + per-step access-pipeline review any future multi-tenant ADR must include.

ADRs are not consensus statements — they record decisions and the reasoning, including the alternatives that were rejected. Disagreement is fine; it goes in the *Consequences* section.

## Releases

Release cadence and tag conventions live in [`docs/RELEASING.md`](docs/RELEASING.md). In short:

- **Tag pushes** trigger [`.github/workflows/release.yml`](.github/workflows/release.yml), which re-runs CI against the tagged SHA before building and publishing GHCR images.
- **Pre-release tags** (`vX.Y.Z-rcN`) publish images at the rc tag but **not** `:latest`, and the GitHub Release is marked pre-release.
- **Stable tags** (`vX.Y.Z`) publish `:vX.Y.Z` AND `:latest`.

Per [`docs/API_STABILITY.md`](docs/API_STABILITY.md): v0.x allows breaking changes between minor versions; v1.0 introduces semver and `/v1/...` versioned endpoints.

Patch releases (`0.1.1`, `0.1.2`, …) **must remain backward-compatible** with their minor base.

Release authority sits with maintainers. Hotfix branching from a tag is documented in the post-release section of the release plan.

## Optional modules

Some surfaces (Second Brain, GraphRAG, alternative webhook adapters) are feature-flag-gated optional modules per [`docs/OSS_V1_SCOPE.md`](docs/OSS_V1_SCOPE.md). PRs adding new optional modules must:

1. Document the enabling env flag(s) in `.env.example` and [`docs/CONFIG_ENV.md`](docs/CONFIG_ENV.md).
2. Default to **disabled** — `make test` and `bash scripts/smoke-local.sh` must pass with the flag off.
3. Add an entry to the *Optional modules* section of `docs/OSS_V1_SCOPE.md`.

Removing an optional module from the OSS scope requires an ADR.

## Code of conduct

All participation is governed by [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md). Maintainers may close threads, lock issues, or block contributors who violate it. Enforcement decisions happen in private and may be appealed via the channel listed in `CODE_OF_CONDUCT.md`.

## Security

Vulnerability reports follow [`SECURITY.md`](SECURITY.md). Security fixes are prioritised over feature work and may ship as an out-of-cycle patch release.

## Changing this document

`GOVERNANCE.md` itself changes via a normal PR with maintainer approval. Substantive changes (adding a role, changing the merge gate, altering release authority) should land alongside an ADR explaining the rationale.

## Related documents

- [`CONTRIBUTING.md`](CONTRIBUTING.md) — how to write a PR.
- [`SUPPORT.md`](SUPPORT.md) — where to ask for help.
- [`docs/DOCS_MAINTENANCE_POLICY.md`](docs/DOCS_MAINTENANCE_POLICY.md) — the doc-impact rule that gates merges.
- [`docs/adr/`](docs/adr/) — ADR index.
- [`docs/RELEASING.md`](docs/RELEASING.md) — release flow.
