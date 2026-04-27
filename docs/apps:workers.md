Asynchronous execution environment.

Suggested responsibilities:

- ingestion runs
- parsing and normalization
- indexing
- chunking and embeddings
- knowledge job execution
- stale checks
- notifications
- maintenance and backfills

## Layout (v1)

The worker binary is part of the **same Go module** as the API (`apps/api`) because Go `internal/` packages cannot be imported from a separate module.

- **Entrypoint:** `apps/api/cmd/jobworker` — Asynq server for `knowledge:job_run` (and future task types).
- **Runbook:** `apps/workers/README.md` — how to start the process and required env vars.

Domain logic stays in `apps/api/internal/<bounded_context>/`; the worker only wires `app.NewDeps` and dispatches tasks.

Worker guidance:

- Workers should call domain services, not reinvent business rules.
- Long-running workflows should be traceable.
- Use stable run IDs and idempotency patterns.
- Partial success should be recordable.
