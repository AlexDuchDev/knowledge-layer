# Grafana dashboards

Reference dashboards for Knowledge Layer (Phase 4.3). Import into your Grafana instance and adapt to your folder/permissions structure.

The dashboards target the metrics surface documented in [docs/ALERTING_PLAYBOOK.md](../../docs/ALERTING_PLAYBOOK.md) — everything `GET /metrics` exposes today (Phase 2.2.2). They assume a Prometheus datasource scrapes the API and that worker health is observed at the worker `/ops/health` endpoints (`:9001`, `:9002`).

## Files

| File | Audience | Panels |
|---|---|---|
| [`knowledge-layer-overview.json`](knowledge-layer-overview.json) | Operators (one screen, "is everything OK?") | API request rate, 5xx rate by route, request duration p95, knowledge-job p95 by job_type, connector sync p95 by adapter_kind, Postgres pool utilisation, Asynq queue depths, Go runtime (goroutines + heap). |

Add more dashboards as you grow — keep this README's table in sync.

## Import

### Via Grafana UI

1. Grafana → Dashboards → Import.
2. Upload the JSON file or paste its contents.
3. Pick your Prometheus datasource when prompted (the dashboards use a `datasource` variable so they work across instances).
4. Save into the folder of your choice.

### Via Grafana API

```bash
curl -s -X POST \
  -H "Authorization: Bearer $GRAFANA_API_TOKEN" \
  -H "Content-Type: application/json" \
  --data @knowledge-layer-overview.json \
  https://grafana.example.com/api/dashboards/db
```

### Via `kube-prometheus-stack` (sidecar import)

If you run the kube-prometheus-stack Helm chart with the `grafana.sidecar.dashboards` sidecar enabled, drop these files into a ConfigMap labeled `grafana_dashboard=1` and the sidecar imports them automatically:

```bash
kubectl -n monitoring create configmap kl-dashboards \
  --from-file=knowledge-layer-overview.json
kubectl -n monitoring label configmap kl-dashboards grafana_dashboard=1
```

## Customisation pointers

Once imported, you'll usually want to:

- **Adjust the scrape job name.** The dashboards default to `job="knowledge-api"`; rename via the dashboard variable if your scrape config uses a different label.
- **Add per-tenant slicing — but don't.** Knowledge Layer is single-tenant ([ADR-0014](../../docs/adr/0014-single-tenant-deployment-stance.md)); there are no `tenant` labels to split by. If you run multiple instances, give each its own scrape job + folder.
- **Wire alerts from the same panels.** The [alerting playbook](../../docs/ALERTING_PLAYBOOK.md) lists the seven alerts you almost certainly want — every threshold matches a panel here.
- **Shorten retention if disk-bound.** All metrics are scrape-time gauges or histograms; default Prometheus 15-day retention is plenty for most Knowledge Layer deployments.

## What's NOT here

- **Per-job-run drill-down dashboards.** Use `audit_events` and `/ops/failed-runs` for forensics on a specific failure — Prometheus is a trends-and-thresholds layer.
- **LLM cost attribution.** Use your provider's dashboard (OpenAI / OpenRouter); the API does not export token-cost metrics.
- **Trace exploration.** Trace export is a Phase 4 follow-up — see [docs/ALERTING_PLAYBOOK.md](../../docs/ALERTING_PLAYBOOK.md) "What's NOT here".

## Maintenance

When the metric surface in [`apps/api/internal/platform/metrics/metrics.go`](../../apps/api/internal/platform/metrics/metrics.go) changes:

1. Update or add panels in the dashboard JSON.
2. Update the table at the top of this README.
3. Update [docs/ALERTING_PLAYBOOK.md](../../docs/ALERTING_PLAYBOOK.md) — alerts and panels usually move together.
