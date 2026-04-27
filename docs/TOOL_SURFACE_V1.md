# TOOL_SURFACE_V1.md

This document specifies the **v1 governed tool surface** for agents and automation clients.

It is the concrete “action space” layer inspired by systems like GBrain, but implemented with Knowledge Layer’s core constraints:

- **permissions-first** (scope before retrieval / action)
- **fail closed**
- **auditability by default**
- **AI is not an authority layer**

The tool surface is intentionally **small and stable**. It maps to existing HTTP routes and internal services, and it is designed to be usable via:

- an internal **Tool Gateway HTTP API** (first implementation), and
- a future external MCP server that mirrors the same contracts.

---

## 1. Design principles

1. **Tools are not “free-form prompts.”** Each tool has explicit schema and explicit permission gates.
2. **No post-filtering of hidden data.** Tool execution must only read within the caller’s allowed scope.
3. **Every tool call is auditable.** Inputs are logged in redacted form; outputs are traceable to governed objects.
4. **Tools never return secrets.** Connector credentials and secret refs are never returned to non-admins; raw content is separately permissioned.
5. **Derived outputs are governed artifacts.** Any created artifact must carry `domain_id`, `owner`, `truth_mode`, `approval_status`, and provenance.

---

## 2. Transport and envelope

### 2.1 Tool Gateway HTTP

Single endpoint:

- `POST /tools/call`

Request envelope:

```json
{
  "tool": "search",
  "args": { "query": "postgres migration plan", "limit": 10 }
}
```

Response envelope (success):

```json
{
  "tool": "search",
  "ok": true,
  "result": { },
  "trace": {
    "tool_call_id": "uuid",
    "started_at": "2026-04-15T12:34:56Z",
    "finished_at": "2026-04-15T12:34:56Z",
    "audit_event_id": "uuid",
    "permission_scope": {
      "domain_ids": ["..."],
      "entity_type_scope": ["..."],
      "sensitivity_cap": 2
    }
  }
}
```

Response envelope (error):

```json
{
  "tool": "sourceFeed.sync",
  "ok": false,
  "error": {
    "code": "forbidden",
    "message": "principal lacks manage_source_feed",
    "details": { "resource_type": "source_feed", "resource_id": "..." }
  },
  "trace": {
    "tool_call_id": "uuid",
    "audit_event_id": "uuid"
  }
}
```

### 2.2 Authentication

Tool Gateway uses the same auth as the API:

- `AUTH_MODE=development_header` (local only) or
- `AUTH_MODE=session` (staging/production)

See: `docs/ACCESS_MODEL.md`.

---

## 3. Tool taxonomy (v1)

### 3.1 Retrieval tools (read-only)

#### Tool: `search`

Purpose: permission-scoped hybrid search across governed entities.

Args:

```json
{
  "query": "string (required)",
  "limit": "number (optional, default 10, max 50)",
  "offset": "number (optional, default 0)",
  "domain_id": "string (optional)",
  "entity_type": "string (optional)",
  "scenario_code": "string (optional)"
}
```

Permission gates:

- **Must be authenticated**.
- Retrieval must be scoped to caller’s granted domains, entity-type scope, sensitivity cap, and object-level ACL.
- If `scenario_code` is non-empty, apply the scenario gate described in `docs/AI_RETRIEVAL_GOVERNANCE.md` (§4.1).

Result (shape):

```json
{
  "hits": [
    {
      "entity_id": "string",
      "entity_type": "string",
      "title": "string",
      "snippet": "string (optional)",
      "domain_id": "string",
      "truth_mode": "canonical_in_platform|mirrored_authority|derived_artifact",
      "approval_status": "not_required|pending_review|approved|rejected",
      "freshness_status": "fresh|review_due|stale|unknown"
    }
  ],
  "total": "number"
}
```

Audit event:

- `tool.search` with `metadata_json.query_redacted`, `result_count`, `latency_ms`, optional `scenario_code` (not sensitive).

Failure modes:

- `403` if `scenario_code` is provided and principal lacks scenario binding.

---

#### Tool: `ask`

Purpose: scoped Q&A with evidence-backed citations and answer traces.

Args:

```json
{
  "question": "string (required)",
  "retrieval_mode": "keyword_only|semantic_only|hybrid (optional)",
  "domain_id": "string (optional)",
  "entity_type": "string (optional)",
  "scenario_code": "string (optional)"
}
```

Permission gates:

- Same as `search` plus scenario gate when `scenario_code` is non-empty.

Result:

```json
{
  "answer_markdown": "string",
  "citations": [
    { "entity_id": "string", "title": "string" }
  ],
  "trust": {
    "partial_due_to_access_limits": "boolean",
    "uses_derived_artifacts": "boolean",
    "freshness_warnings": ["string"]
  },
  "answer_trace_id": "string"
}
```

Audit event:

- `tool.ask` with `metadata_json.question_redacted`, `retrieval_mode`, `answer_trace_id`.

---

#### Tool: `entity.get`

Purpose: fetch one entity (governed object) by id.

Args:

```json
{ "entity_id": "string (required)" }
```

Permission gates:

- Must pass `view` for the entity via `identity_access.AccessEvaluator` / permissions resolver.

Result: entity detail payload (same as `GET /entities/:id`).

Audit event:

- `tool.entity_get` with `entity_id`.

---

#### Tool: `entity.related`

Purpose: bounded relation-aware expansion.

Args:

```json
{
  "entity_id": "string (required)",
  "depth": "number (optional, default 1, max 2)"
}
```

Permission gates:

- Must pass `view` for the starting entity.
- Every expanded related entity must independently pass `view` (fail closed per target).

Result:

```json
{
  "nodes": [{ "entity_id": "string", "title": "string", "entity_type": "string" }],
  "edges": [{ "from": "string", "to": "string", "relation_type": "string" }]
}
```

Audit event:

- `tool.entity_related` with `entity_id`, `depth`, and counts.

---

### 3.2 Knowledge job tools

#### Tool: `job.run`

Purpose: start a knowledge job run (async).

Args:

```json
{ "job_id": "string (required)" }
```

Permission gates:

- Same gate as `POST /knowledge-jobs/:id/run` (owner or operator binding or domain permission) per `docs/ACCESS_MODEL.md`.

Result:

```json
{ "job_run_id": "string", "status": "queued|running" }
```

Audit event:

- `tool.job_run` with `job_id`, `job_run_id`.

---

#### Tool: `job.status`

Purpose: fetch status + summary for a job run.

Args:

```json
{ "job_run_id": "string (required)" }
```

Permission gates:

- Must be allowed to view the job’s output domain (same as jobs UI access).

Result: job run detail payload (same as `GET /job-runs/:id`).

Audit event:

- `tool.job_status` with `job_run_id`.

---

### 3.3 Connector / ingestion ops tools (admin + governance)

These tools are **operator/admin** surfaces. They never bypass feed governance.

#### Tool: `sourceFeed.list`

Args:

```json
{ "domain_id": "string (optional)", "limit": "number (optional)", "offset": "number (optional)" }
```

Permission gates:

- Domain-scoped list based on caller’s grants; sensitive connector config omitted unless privileged (existing behavior).

Audit event:

- `tool.source_feed_list`.

---

#### Tool: `sourceFeed.sync`

Args:

```json
{ "source_feed_id": "string (required)" }
```

Permission gates:

- Must pass `manage_source_feed` for the feed’s domain (existing behavior in connector framework).

Audit event:

- `tool.source_feed_sync` with `source_feed_id` and outcome.

---

#### Tool: `rawArtifact.get`

Args:

```json
{ "raw_artifact_id": "string (required)" }
```

Permission gates:

- Must pass **view_raw** (or equivalent) as implemented in the API for raw artifacts; never fall back to domain view.

Audit event:

- `tool.raw_artifact_get` with `raw_artifact_id`.

---

## 4. Required audit + trace fields

For every tool call, persist:

- `tool_call_id` (uuid)
- `tool_name`
- `principal_user_id` (and team if relevant)
- `request_redacted_json`
- `started_at`, `finished_at`
- `outcome` (`success|error|denied`)
- `error_code` (if any)
- **supporting ids** for explainability:
  - `entity_ids` touched
  - `source_feed_id` / `raw_artifact_id` where applicable
  - `answer_trace_id` for `ask`
  - `job_id` / `job_run_id` for jobs

This should be represented as an audit event row plus an optional dedicated tool-call table if needed (implementation detail).

---

## 5. Redaction rules (mandatory)

Tool Gateway must **redact** before writing audit metadata:

- user-provided text fields: `query`, `question`, freeform filters → store hashed + truncated preview (or privacy-sanitized representation) depending on `AI_PRIVACY_FLOW.md`.
- connector configs, secret refs, OAuth codes → never store raw; store presence booleans only.

---

## 6. Backward compatibility

The tool surface wraps existing HTTP routes. It does not replace them.

This allows:

- Web UI to keep using first-party endpoints.
- Agents to use a stable tool contract with uniform tracing/audit.

---

## 7. Versioning

The Tool Gateway must expose a version field in responses and accept:

- `X-Tool-Surface-Version: v1` (optional; default `v1`)

Breaking changes require `v2` and a migration window.

