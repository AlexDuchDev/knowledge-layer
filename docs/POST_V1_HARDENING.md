# Post–v1 OSS hardening checklist

After [OSS_V1_SCOPE.md](./OSS_V1_SCOPE.md) is stable and staging/production runbooks are green, use this list for **cost, retention, and observability** work. Items are intentionally high-level; each should become a ticket with owner and metric.

## Blob and raw retention

- Replace default `blobstore.Nop` with an S3/R2-compatible adapter where raw payloads must survive container restarts ([LIMITATIONS.md](./LIMITATIONS.md)).
- Define retention policy (time + legal hold) per domain or source feed class; document operator toggles.

## AI and embedding cost controls

- Track embedding token volume / request counts per workspace (Prometheus counters or provider billing export).
- Add rate limits or queue back-pressure on `retrieval:embed_chunk` if a tenant runs hot connectors.
- Revisit [adr/0013-embeddings-privacy-boundary.md](./adr/0013-embeddings-privacy-boundary.md) if optional pre-embed redaction ships.

## Observability SLOs

- Define SLOs for: API availability, `POST /ask` latency (p95), search error rate, job run failure rate.
- Wire alerts (PagerDuty / email) on SLO burn; keep `GET /metrics` + `OPS_AUTH_TOKEN` pattern in production ([SELF_HOSTED.md](./SELF_HOSTED.md)).
- Starter table: [SLO_AND_ALERTING_TEMPLATE.md](./SLO_AND_ALERTING_TEMPLATE.md).

## Performance at scale

- OpenSearch index sizing, Postgres vacuum/index cadence, Neo4j (if enabled) disk growth.
- Load tests on hybrid retrieval under representative domain counts.

## Related

- [RUNBOOK_STAGING_PROD.md](./RUNBOOK_STAGING_PROD.md)
- [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md)
