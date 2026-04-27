# Knowledge jobs engine

**Canonical:** [knowledge-jobs-engine.md](knowledge-jobs-engine.md), [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md), [job-builder.md](job-builder.md).

**Code:** `apps/api/internal/knowledge_jobs/`, worker `apps/api/cmd/jobworker`.

Jobs are **first-class governed objects** with triggers, scope, and metrics. Orchestration runs in the worker; API enqueues and surfaces status.

When job types or orchestration change, update the canonical docs and [API_SURFACE_V1.md](API_SURFACE_V1.md) if routes change.
