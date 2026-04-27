# Workers (background jobs)

Worker binaries live in the **same Go module** as the API (`apps/api`) because Go `internal/` rules forbid a separate module from importing `github.com/knowledgelayer/api/internal/...`.

```bash
cd apps/api && go run ./cmd/jobworker
cd apps/api && go run ./cmd/connectorworker
```

- **jobworker:** `knowledge:job_run`, `knowledge:scheduled_tick` (cron-like dispatch), and `secondbrain:outbound_delivery` (only when SecondBrain optional module is enabled). Needs `DATABASE_URL`, `REDIS_URL`.
- **connectorworker:** `connector:source_sync` (adapter polling), `ingestion:process_artifact` (raw → normalized → chunks + audit), `retrieval:embed_chunk` (embed + upsert), `graphrag:extract_entity` (when Neo4j is configured). Needs `DATABASE_URL`, `REDIS_URL` (and `NEO4J_URL` for graph extraction).

When `REDIS_URL` is unset, the API runs applicable job work synchronously where implemented.

See [docs/backend-architecture.md](../docs/backend-architecture.md) and [docs/permission-flow.md](../docs/permission-flow.md).