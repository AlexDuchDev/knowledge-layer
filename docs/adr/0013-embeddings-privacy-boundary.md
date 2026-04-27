# ADR 0013: Embeddings vs PrivacyGateway boundary

**Status:** Accepted  
**Date:** 2026-04-21  

## Context

`PrivacyGateway` (sanitization, vault, rehydration, policy) applies to **generative** OpenAI chat paths used for Ask and HTTP AI helpers ([AI_PRIVACY_FLOW.md](../AI_PRIVACY_FLOW.md)).  
**Embedding** calls (`POST …/embeddings` via `internal/llm/embed.go` and `connectorworker` retrieval embed tasks) send **chunk text** that was already ingested under workspace/source-feed policy. They do not carry end-user prompts in the same way Ask does.

Routing embeddings through the full gateway would add latency, complexity, and unclear benefit unless chunk text is treated as equivalently sensitive to assembled Ask context.

## Decision

1. **Embeddings remain outside `PrivacyGateway` for v1**, with the following **explicit guarantees**:
   - Only **chunk rows** that exist in Postgres after ingestion/normalization are embedded (no ad-hoc user paste into the embed path).
   - Ingestion and chunk creation remain subject to **access-before-retrieval** and connector governance ([0002-access-before-retrieval.md](./0002-access-before-retrieval.md)).
   - Embedding transport uses the same **OpenAI-compatible** env as chat (`OPENAI_*` / `OPENROUTER_*`); operators must treat provider logs like any other subprocessed data (see [SELF_HOSTED.md](../SELF_HOSTED.md)).

2. **Any new OpenAI call that consumes user-typed or cross-tenant context** (chat, audio-in, tool output shown to users, **future job LLM steps**) **must** go through `PrivacyGateway` or an ADR that narrows the exception.

3. **Follow-up (non-blocking for OSS v1):** optional **pre-embed redaction** hook (same pattern library as sanitization) if a deployment classifies chunk bodies as highly sensitive; tracked as product backlog, not default behavior.

## Consequences

- Docs and `LIMITATIONS` must state clearly that **“AI privacy gateway” ≠ all OpenAI traffic**.
- Security reviews focus on **ingestion boundary + chunk ACL + Ask path**, not on pretending embeddings are gateway-covered today.

## Related

- [AI_PRIVACY_FLOW.md](../AI_PRIVACY_FLOW.md) §6  
- `apps/api/internal/llm/embed.go`, `apps/api/cmd/connectorworker/main.go` (retrieval embed task)
