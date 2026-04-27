# Deployment

Operator-facing **hub** for running the platform beyond local dev. Follow linked documents; avoid duplicating checklists here.

## Self-hosted and operations

- [SELF_HOSTED.md](SELF_HOSTED.md) — instance model and bootstrap.
- [OPERATIONS.md](OPERATIONS.md) — day-2 operations.

## Checklists and hardening

- [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md) — services, env, order, compose.
- [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md) — staging/production rules.
- [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) — go/no-go.
- [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md) — smoke steps.

## Configuration

- [CONFIG_ENV.md](CONFIG_ENV.md), [Config and environments.md](Config%20and%20environments.md)
- Root [`.env.example`](../.env.example)

## Cloud: DigitalOcean

- [DO_DEPLOYMENT.md](DO_DEPLOYMENT.md) — DO App Platform / droplet notes.
- [DO_INFRA_TOPOLOGY.md](DO_INFRA_TOPOLOGY.md) — suggested topology.

## Examples

- [SETUP_EXAMPLES.md](SETUP_EXAMPLES.md) — env and topology examples.
- [EXAMPLES.md](EXAMPLES.md) — product and API examples.

When **runtime or deploy behavior** changes, update the relevant checklist and `.env.example` ([DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md)).
