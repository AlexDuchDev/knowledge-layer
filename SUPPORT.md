# Getting help

Knowledge Layer is a self-hosted, single-tenant project maintained by volunteer contributors. There is no commercial support tier. This page tells you which channel matches which kind of question and what response time to expect.

## Which channel for what

| Question | Where it goes | Why |
|---|---|---|
| **You found a bug** | [GitHub Issues](../../issues) → *Bug report* template | Reproducible defects with steps + environment so a maintainer (or you) can fix them. |
| **You want a new feature** | [GitHub Issues](../../issues) → *Feature request* template | The template walks you through the five-question governance checklist (access / audit / privacy / provenance / failure-mode). Triage uses those answers. |
| **A doc is wrong, missing, or unclear** | [GitHub Issues](../../issues) → *Docs improvement* template, **or** open a PR directly | Direct PRs land faster than issues for typo / link / wording fixes. |
| **You have a question about how to use it** | GitHub Discussions (when enabled), otherwise the *Docs improvement* issue template | If the answer should live somewhere documented, it almost always should — flagging it as a docs gap closes the loop for the next person. |
| **You suspect a security vulnerability** | [`SECURITY.md`](SECURITY.md) — private GitHub advisory flow | **Do not** open a public issue with exploit details. See SECURITY.md for the disclosure path. |

Before filing anything, please:

1. Search existing [issues](../../issues) and [discussions](../../discussions) — your question may already be answered.
2. Check [`docs/LIMITATIONS.md`](docs/LIMITATIONS.md) — some behaviours are intentionally stubbed or feature-flagged.
3. Check [`docs/OSS_V1_SCOPE.md`](docs/OSS_V1_SCOPE.md) — confirm the area you're asking about is in scope for the OSS release.

## Response time expectations

This is a community-maintained project. Best-effort targets:

- **Bug reports**: first triage response within ~48 business hours.
- **Feature requests**: first response within ~1 week, often as "added to backlog" rather than an immediate decision.
- **Security reports**: acknowledgement within 24 hours per [`SECURITY.md`](SECURITY.md).
- **Discussions / Q&A**: no SLA; checked at least weekly.

If a thread goes silent, a polite bump after a week is welcome.

## Things that will get a faster answer

- A small reproduction repo or a `curl` command that demonstrates the issue.
- The output of `bash scripts/smoke-local.sh` or `bash scripts/smoke-session.sh`.
- The output of `GET /audit-events` filtered to the failing flow, and `GET /ops/failed-runs` for ingestion / job failures.
- Your `APP_ENV`, `AUTH_MODE`, and the image tag or commit SHA you're running.
- Whether you're on docker compose, Kubernetes ([`deploy/k8s/`](deploy/k8s/)), or bare-metal / systemd ([`docs/SELF_HOSTED.md`](docs/SELF_HOSTED.md)).

## Things that probably will not get an answer here

- Requests for hands-on consulting, custom-deployment help, or paid support.
- "Will you build feature X for my company?" — file a feature request and we'll discuss; we will not commit roadmap changes via support threads.
- Re-opening closed issues without new information.

## Related documents

- [`CONTRIBUTING.md`](CONTRIBUTING.md) — how to land a PR.
- [`GOVERNANCE.md`](GOVERNANCE.md) — how decisions get made.
- [`docs/PRIVACY_AND_TELEMETRY.md`](docs/PRIVACY_AND_TELEMETRY.md) — what the project does and does not collect.
- [`docs/RELEASING.md`](docs/RELEASING.md) — release cadence and tag conventions.
