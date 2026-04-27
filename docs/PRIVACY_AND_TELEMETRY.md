# Privacy and telemetry

## TL;DR

**Knowledge Layer collects no usage analytics.** No phone-home, no ping endpoint, no anonymous metrics, no crash reporting to a third party. Operators own their observability stack end-to-end. This is a deliberate design choice, aligned with the single-tenant deployment stance ([ADR-0014](adr/0014-single-tenant-deployment-stance.md)) and the production-hardening posture ([PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md)).

This page documents what the project does and does not do with respect to outbound data, so operators auditing the deployment have a single authoritative reference.

## What the project does NOT do

The following are **not** present anywhere in the codebase. Adding any of them requires a new ADR that explicitly supersedes this document.

| Anti-feature | Status |
|---|---|
| Anonymous usage statistics ("X instances are running, Y MAU") | Not present. No code path emits to an upstream collector. |
| Crash / error reporting to a third party (Sentry, Bugsnag, Rollbar, etc.) | Not present. Errors land in your logs and `audit_events`; nothing leaves the instance. |
| Product analytics SDKs (Segment, Mixpanel, PostHog, Amplitude, Heap, GA4) on the web app | Not present. The Next.js app ships no analytics scripts. |
| Update checks / version pings | Not present. The instance never reaches out to "check for new versions". |
| Marketing pixels, tag managers, ad-network beacons | Not present. |
| Feature-flag SaaS (LaunchDarkly, ConfigCat, etc.) phoning home | Not present. Optional modules use plain env-var gates documented in [`CONFIG_ENV.md`](CONFIG_ENV.md). |

## What the project DOES emit (operator-controlled)

The instance emits observability data **only when you configure it to**, and **only to endpoints you control**:

| Surface | Where it goes | How to enable / disable |
|---|---|---|
| **Application logs** (stdout/stderr from API + workers) | wherever your container runtime / systemd / journald sends them | always on; configure log retention via your runtime |
| **`GET /metrics`** — Prometheus text/OpenMetrics (HTTP request counts, job + connector durations, Asynq queue depths, Postgres pool stats, Go runtime) | the Prometheus you point at the instance | bearer-gated by `OPS_AUTH_TOKEN` outside `APP_ENV=local` ([routes_health.go](../apps/api/internal/httpserver/routes_health.go)) |
| **`GET /ops/health`** on workers | your monitoring | same bearer gate |
| **`audit_events` table** in your Postgres | your Postgres | always on; this is the audit trail, not telemetry |
| **Outbound LLM calls** (when you configure an LLM provider) | the LLM provider you configured | only fires when you set the provider env vars; routed through `ai/privacy.PrivacyGateway` so PII is sanitized first per [AI_PRIVACY_FLOW.md](AI_PRIVACY_FLOW.md) |
| **Outbound webhook deliveries** (Slack, Mattermost, future adapters) | the URL the operator configured per source feed | only fires for source feeds the operator has wired up |
| **Outbound connector polls / API calls** (Google Workspace, Notion, etc., when configured) | the SaaS endpoint of the connector | per-feed; documented in [CONNECTOR_FRAMEWORK.md](CONNECTOR_FRAMEWORK.md) |

The `/metrics` endpoint is **scrape-pulled** by your Prometheus, not pushed by the instance. The instance has no concept of "where to send metrics" — it only exposes them.

## Why this stance

1. **Operators run this on their own infrastructure with their own data.** Outbound telemetry from a single-tenant memory platform is a privacy and compliance liability that the project has no business taking on.
2. **The audit trail already lives in Postgres.** `audit_events` records every access-sensitive operation. Operators query it directly; the project does not need a parallel pipeline to learn what's happening.
3. **Observability is a deployment concern, not a product concern.** Per [ADR-0014](adr/0014-single-tenant-deployment-stance.md), the project does not assume a shared control plane. Each operator owns their Prometheus, Grafana, log aggregation, alerting — see [`docs/ALERTING_PLAYBOOK.md`](ALERTING_PLAYBOOK.md) and [`deploy/grafana/`](../deploy/grafana/) for starting points.
4. **Trust beats inference.** Operators can audit the codebase and verify that no telemetry exists. A "but it's anonymized" promise costs more credibility than the data would be worth.

## How the project learns about adoption

Without telemetry, the project learns about adoption only through:

- GitHub stars and forks (passively visible via the GitHub UI).
- Issues, discussions, and PRs that operators choose to file.
- Direct outreach when operators choose to share.

This is intentional. If you want to share that you're using the project, please [file an issue or discussion](../../issues) — we'd love to hear about your deployment, what worked, and what didn't.

## Data the operator collects from end users

Knowledge Layer ingests organisational content via connectors and stores it under operator control. That data is:

- Stored exclusively in **your** Postgres / Redis / OpenSearch.
- Never relayed to the project, the maintainers, or any third party by the application code.
- Subject to whatever retention, deletion, and export policies the operator configures (see [`docs/OPERATIONS.md`](OPERATIONS.md)).
- Routed through `ai/privacy.PrivacyGateway` before any outbound LLM call — placeholder cleartext is encrypted at rest with `AI_PRIVACY_VAULT_KEY` ([PRODUCTION_HARDENING.md §2](PRODUCTION_HARDENING.md), [AI_PRIVACY_POLICY.md](AI_PRIVACY_POLICY.md)).

If the operator configures an LLM provider, prompts and (sanitized) context are sent to that provider per the provider's own data-handling policy. The project does not see that traffic.

## Cookies, sessions, and the web app

When `AUTH_MODE=session`, the web app sets a single signed first-party cookie (`kl_session`) for authentication. There are no third-party cookies, tracking pixels, or cross-origin beacons set by the application. The cookie is `Secure` in staging/production per [PRODUCTION_HARDENING.md §1](PRODUCTION_HARDENING.md).

In `AUTH_MODE=development_header` (local pilot only), no cookie is set; the `X-Principal-User-ID` header is supplied by the developer.

## Changing this stance

Adding any form of outbound telemetry requires:

1. A new ADR explicitly superseding this document, with the rationale, the data shape, the opt-out mechanism, and the privacy review.
2. An update to [LIMITATIONS.md](LIMITATIONS.md) and the README so operators are not surprised.
3. Default-off behaviour, per the optional-modules contract in [`OSS_V1_SCOPE.md`](OSS_V1_SCOPE.md).

Until that ADR exists and is Accepted, the answer to "should we add anonymous pings?" is **no**.

## Related documents

- [ADR-0014: Single-tenant deployment stance](adr/0014-single-tenant-deployment-stance.md)
- [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) — the bearer-gate rules for `/metrics` and `/ops/health`
- [AI_PRIVACY_POLICY.md](AI_PRIVACY_POLICY.md) — how the privacy gateway sanitizes LLM-bound traffic
- [AI_PRIVACY_FLOW.md](AI_PRIVACY_FLOW.md) — the encrypt-then-rehydrate pattern
- [ALERTING_PLAYBOOK.md](ALERTING_PLAYBOOK.md) — operator-side observability starting point
- [SELF_HOSTED.md](SELF_HOSTED.md) — single-tenant deployment guide
