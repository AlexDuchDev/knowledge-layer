# OpenSearch: development vs production

## Development (this repository)

The default [docker-compose.yml](../docker-compose.yml) `opensearch` service sets:

- `discovery.type=single-node`
- `plugins.security.disabled=true`

This keeps local onboarding fast and matches [LIMITATIONS.md](./LIMITATIONS.md) (“OpenSearch (local compose) — security disabled — **dev only**”).

**Do not** copy these settings into internet-exposed production without replacing them with a secured OpenSearch deployment (TLS, authentication, least-privilege network).

## Production expectations

1. **Enable the security plugin** (or place OpenSearch behind a private network + authenticated reverse proxy).
2. **TLS** between API/workers and OpenSearch; rotate credentials via secrets manager.
3. **Separate clusters** for staging and production; never reuse admin certificates across environments.
4. Set `OPENSEARCH_URL` in the API to the **HTTPS** endpoint the secured cluster exposes.

## Degraded mode

If `OPENSEARCH_URL` is empty, search degrades to DB-backed behavior (see [SELF_HOSTED.md](./SELF_HOSTED.md) capability matrix). Access control still applies; only recall changes.

## Related

- [RUNBOOK_STAGING_PROD.md](./RUNBOOK_STAGING_PROD.md)
- [STAGING_SMOKE_TEST.md](./STAGING_SMOKE_TEST.md) § dependency health
