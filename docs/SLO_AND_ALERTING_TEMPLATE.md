# SLO and alerting template (post–v1)

Use this after initial production stability ([POST_V1_HARDENING.md](./POST_V1_HARDENING.md)). Copy rows into your monitoring tool; thresholds are examples only.

| SLI | Example SLO | Alert (example) |
|-----|----------------|-----------------|
| API availability | 99.5% monthly from `/health` or synthetic probe | Page if 5m error budget burn |
| `POST /ask` latency | p95 < 30s (tune to model + corpus) | Warn on sustained p95 regression |
| Search error rate | < 1% 5xx on `/search` | Spike vs 24h baseline |
| Job run failures | `knowledge:job_run` DLQ / failure ratio | Queue depth + failure counter |
| DB / Redis / OpenSearch | Dependency OK in `/ops/health` (authenticated) | Same signal as runbook |

**Metrics source:** `GET /metrics` with `OPS_AUTH_TOKEN` bearer ([PRODUCTION_GO_LIVE_CHECKLIST.md](./PRODUCTION_GO_LIVE_CHECKLIST.md) §9).

**Cost / AI:** track embedding and chat token usage per provider billing; add internal counters when adopting [POST_V1_HARDENING.md](./POST_V1_HARDENING.md) § AI cost controls.
