# Alerting playbook

What to alert on, and what each alert means. This is a **reference** — pick the rules that match your SLOs and route them to your on-call platform (PagerDuty, OpsGenie, Slack, …).

The metrics referenced below are the ones Knowledge Layer exposes at `GET /metrics` (Phase 2.2.2). Before you wire anything: confirm `OPS_AUTH_TOKEN` is set on the API and that your Prometheus scrape config sends `Authorization: Bearer …`.

## TL;DR — alerts you almost certainly want

| Severity | Trigger | Why it matters |
|---|---|---|
| **page** | `up{job="knowledge-api"}` is 0 for 2m | API is down, no users can sign in or query. |
| **page** | `(asynq_queue_active{queue=~"connectors\|ingestion"} == 0) and (asynq_queue_pending{queue=~"connectors\|ingestion"} > 0)` for 10m | Worker is alive but consuming nothing — ingestion has stalled. |
| **page** | `rate(http_requests_total{route!~"/health\|/metrics",code=~"5.."}[5m]) > 0.1` for 5m | API is 5xx-ing real traffic. |
| **ticket** | `asynq_queue_failed{queue=~".+"} > 50` for 1h | Failed-task buffer is filling; investigate `/ops/failed-runs`. |
| **ticket** | `histogram_quantile(0.95, rate(knowledge_job_run_duration_seconds_bucket[1h])) > 300` | Job p95 jumped past 5 minutes — likely a bad LLM key, slow source, or stuck dependency. |
| **ticket** | `postgres_pool_acquired_conns / postgres_pool_max_conns > 0.85` for 15m | Connection pool saturated; expect intermittent timeouts. |
| **info** | `time() - max(audit_events_created_at) > 3600` (custom) | No audit activity for an hour — either traffic genuinely dropped to zero, or audit emission broke (compliance signal). |

## Metrics reference

What's exposed at `/metrics` on the API (`:8080/metrics`, bearer-gated outside `APP_ENV=local`):

| Metric | Type | Labels | Source |
|---|---|---|---|
| `http_requests_total` | counter | `method`, `route`, `code` | Fiber middleware on every HTTP request. |
| `knowledge_job_run_duration_seconds` | histogram | `job_type`, `status` | `knowledge_jobs.runOrchestrator.execute` after each run (Phase 2.2.2). |
| `connector_sync_duration_seconds` | histogram | `adapter_kind`, `status` | `ingestion_connectors/app.SyncOrchestrator.RunSync` after each sync. |
| `postgres_pool_total_conns` / `_idle_conns` / `_acquired_conns` / `_max_conns` | gauge | — | On-scrape from `pgxpool.Stat()`. |
| `postgres_pool_acquire_count_total` / `_acquire_duration_seconds_total` | counter | — | Cumulative since process start. |
| `asynq_queue_pending` / `_active` / `_scheduled` / `_retry` / `_archived` / `_failed` | gauge | `queue` | On-scrape from Asynq Inspector (only when Redis is configured). |
| Standard Go and process collectors (`go_*`, `process_*`) | mixed | — | `prometheus/client_golang/prometheus/collectors`. |

Worker `/ops/health` (Phase 2.2.1) is NOT a Prometheus surface — it's a JSON snapshot at `:9001/ops/health` (jobworker) and `:9002/ops/health` (connectorworker). Useful for on-demand check or a custom blackbox probe.

## Alert rules (Prometheus)

Drop into your rules file and adapt thresholds. All assume the API is scraped under `job="knowledge-api"` (rename to match your scrape config).

### API liveness

```yaml
groups:
  - name: knowledge-layer-api
    rules:
      - alert: KnowledgeAPIDown
        expr: up{job="knowledge-api"} == 0
        for: 2m
        labels:
          severity: page
        annotations:
          summary: "Knowledge Layer API is down"
          runbook_url: "https://github.com/<org>/knowledge-layer/blob/main/docs/OPERATIONS.md#health"
          description: "Scrape target {{ $labels.instance }} unreachable for 2m. Check pod status, ingress, and API logs."

      - alert: KnowledgeAPI5xxSpike
        expr: |
          sum by (route) (rate(http_requests_total{job="knowledge-api",code=~"5..",route!~"/health|/metrics"}[5m]))
          /
          sum by (route) (rate(http_requests_total{job="knowledge-api",route!~"/health|/metrics"}[5m]))
          > 0.05
        for: 5m
        labels:
          severity: page
        annotations:
          summary: "5xx rate > 5% on {{ $labels.route }}"
          description: "Sustained 5xx rate above 5% for 5m. Check `/ops/failed-runs`, recent deploys, and dependency health."
```

### Worker liveness via queue depth

The worker `/health` HTTP endpoint is a process-liveness probe; it does not detect "process is alive but stuck". Use queue metrics for that:

```yaml
      - alert: KnowledgeWorkerStalled
        expr: |
          (asynq_queue_active{queue=~"connectors|ingestion|retrieval|ai|governance"} == 0)
          and
          (asynq_queue_pending{queue=~"connectors|ingestion|retrieval|ai|governance"} > 0)
        for: 10m
        labels:
          severity: page
        annotations:
          summary: "Worker stalled on queue {{ $labels.queue }}"
          description: |
            No tasks active but pending > 0 for 10m. Worker process likely deadlocked, wedged on a poison message,
            or out of dependencies (Redis, OpenSearch, LLM).
            Check {{ $labels.queue }} worker `/ops/health` last_processed_by_task field
            (jobworker :9001, connectorworker :9002) and recent worker logs.

      - alert: KnowledgeQueueBacklog
        expr: asynq_queue_pending{queue=~"connectors|ingestion|retrieval|ai|governance"} > 1000
        for: 30m
        labels:
          severity: ticket
        annotations:
          summary: "Queue {{ $labels.queue }} has > 1000 pending for 30m"
          description: "Sustained backlog. Either traffic spike or worker capacity insufficient — consider HPA or larger pods."

      - alert: KnowledgeQueueFailedSpiking
        expr: increase(asynq_queue_failed{queue=~".+"}[15m]) > 20
        for: 5m
        labels:
          severity: ticket
        annotations:
          summary: "Failed tasks rising on queue {{ $labels.queue }}"
          description: "20+ task failures in the last 15m. Inspect `/ops/failed-runs` and recent error logs."
```

### Job execution health

```yaml
      - alert: KnowledgeJobRunSlowP95
        expr: |
          histogram_quantile(
            0.95,
            sum by (job_type, le) (rate(knowledge_job_run_duration_seconds_bucket[1h]))
          ) > 300
        for: 30m
        labels:
          severity: ticket
        annotations:
          summary: "Job {{ $labels.job_type }} p95 > 5m"
          description: "Investigate LLM provider latency, source feed size, or governance gate slowness."

      - alert: KnowledgeJobFailureRate
        expr: |
          sum by (job_type) (rate(knowledge_job_run_duration_seconds_count{status="failed"}[1h]))
          /
          sum by (job_type) (rate(knowledge_job_run_duration_seconds_count[1h]))
          > 0.10
        for: 30m
        labels:
          severity: ticket
        annotations:
          summary: "Job {{ $labels.job_type }} failure rate > 10%"
          description: "Sustained failure rate. Check `/ops/failed-runs` filtered by job type."
```

### Connector sync health

```yaml
      - alert: KnowledgeConnectorSyncFailing
        expr: |
          sum by (adapter_kind) (rate(connector_sync_duration_seconds_count{status="failed"}[1h]))
          /
          sum by (adapter_kind) (rate(connector_sync_duration_seconds_count[1h]))
          > 0.20
        for: 30m
        labels:
          severity: ticket
        annotations:
          summary: "Connector {{ $labels.adapter_kind }} failing > 20%"
          description: "Source credentials may be expired or upstream API throttling. Check the feed in /control-plane/sources."
```

### Database pool

```yaml
      - alert: KnowledgePoolSaturated
        expr: postgres_pool_acquired_conns / postgres_pool_max_conns > 0.85
        for: 15m
        labels:
          severity: ticket
        annotations:
          summary: "Postgres pool > 85% utilised"
          description: |
            High pool utilisation; expect timeouts under bursts.
            Either scale the pool (`PGX_MAX_CONNS` if exposed in your build), reduce per-request DB work,
            or scale the API horizontally (more pods = more pools but same DB ceiling).

      - alert: KnowledgePoolExhausted
        expr: postgres_pool_idle_conns == 0 and postgres_pool_acquired_conns >= postgres_pool_max_conns
        for: 5m
        labels:
          severity: page
        annotations:
          summary: "Postgres pool exhausted"
          description: "All connections checked out, none idle. Requests are blocking on acquire — investigate slow queries."
```

### LLM-side anomalies

The API does not directly export LLM latency metrics today (Phase 4 follow-up). For now, use:

- **Outbound HTTP at the egress proxy** (if you have one) for `api.openai.com` 4xx/5xx rate.
- **`http_requests_total{route="/ask",code="500"}`** as a proxy — Ask 500s usually mean the LLM call failed.
- **`knowledge_job_run_duration_seconds_bucket`** — every wired processor calls the LLM, so p95 spikes there often track LLM-side issues.

## Audit observability (separate channel)

Prometheus is for ops. Audit-event monitoring is for compliance / security:

- `vault.placeholder_stored`, `vault.placeholder_decrypted`, `vault.rehydration_applied` (Phase 1.1.3 hardening) appear in `audit_events` for every Ask that uses sensitive content. **Their absence after a known-sensitive Ask** signals a vault wiring problem — not a Prometheus alert, but worth a periodic check.
- `ingestion.artifact_processed` per raw artifact — `outcome="error"` rows correlate with connector failures.
- `entity.asked` per Ask request.

Recommended: a daily job that queries `audit_events` and Slack-pings if any of:
- The vault audit triple is missing for the last 24h while Ask traffic happened.
- `entity.asked` count is dramatically below baseline (silent outage).
- Any `policy_exceptions.*` event fired (review needed).

## Wiring with kube-prometheus-stack

Drop the rules into your operator config:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: knowledge-layer
  namespace: knowledge-layer
spec:
  groups:
    - name: knowledge-layer-api
      rules:
        # paste the rules above here
```

Scrape config (ServiceMonitor):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: knowledge-api
  namespace: knowledge-layer
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: api
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
      bearerTokenSecret:
        name: knowledge-layer-secrets
        key: OPS_AUTH_TOKEN
```

## What's NOT here

- **Trace-based alerts** (Honeycomb / Tempo / OTLP) — Knowledge Layer doesn't yet emit traces; Phase 4 follow-up.
- **Per-tenant alerts** — Knowledge Layer is single-tenant per instance; cross-tenant rate limiting is out of scope (see [docs/adr/](./adr/) for the multi-tenant ADR planned for Phase 4.4).
- **Cost alerts** — LLM usage cost lives in the provider dashboard, not the API.

## Related

- [OPERATIONS.md](./OPERATIONS.md) — health endpoints, failure signals, incident checklist.
- [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md) — `/metrics` and `/ops/health` auth contract.
- [UPGRADE_AND_ROLLBACK.md](./UPGRADE_AND_ROLLBACK.md) — what to expect during deploys (some alerts will briefly fire).
- [SECRET_ROTATION.md](./SECRET_ROTATION.md) — verification checklist after rotation overlaps with what these alerts watch for.
