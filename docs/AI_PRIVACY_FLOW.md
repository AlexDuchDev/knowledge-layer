# AI privacy flow (end-to-end)

This document ties together policy, sanitization, vault, rehydration, and traces for **Ask** and **HTTP AI helpers**. Canonical detail lives in:

- [AI_PRIVACY_POLICY.md](./AI_PRIVACY_POLICY.md)
- [AI_SANITIZATION_LAYER.md](./AI_SANITIZATION_LAYER.md)
- [AI_REHYDRATION_LAYER.md](./AI_REHYDRATION_LAYER.md)

## 1. Ask (`POST /entities/:id/ask`, `POST /ask`)

1. Authenticate and resolve **view** on evidence entities (existing behavior).
2. Build `[]privacy.TextSegment` via `qa` helpers: question + per-entity **header** segments (`SkipPatternDetection`) + **body** segments (patterns allowed).
3. `PrivacyGateway.InvokeOpenAI`:
   - `SensitiveDataPolicyService.Resolve` with `PolicyContext{domain_id, scenario: ask_entity|ask_global, output_type: qa_answer}`.
   - `SanitizationService.SanitizeSegments` → stable placeholders.
   - If vault configured (`AI_PRIVACY_VAULT_KEY` or dev plaintext flag), `SecureEntityMapStore.PutBatch` keyed by **answer trace id**.
   - `Chat` on **sanitized** user payload.
   - `RehydrateFromTokenizer` with `RehydrationMode` partial by default, publication/sensitivity clamps.
4. Parse JSON answer/citations; persist `answer_traces` including **`privacy_json`**.

## 2. HTTP AI helpers (`POST /ai/summarize`, `POST /ai/draft-suggestions`)

Same gateway path with scenarios `ai_summarize` / `ai_draft_suggestions` and output types `summary` / `draft_suggestions`. Draft suggestions pass the draft **entity** for sensitivity and rehydration checks.

## 3. Knowledge jobs

`weekly_digest` today **does not** call an LLM; it aggregates `normalized_records` into a derived entity. **Any future job processor that calls a completion model must use `PrivacyGateway`** with `job_type`, `output_type`, `job_run_id`, and correlation id = run id.

## 4. Environment

| Variable | Purpose |
|----------|---------|
| `AI_PRIVACY_VAULT_KEY` | 32-byte AES key (raw or base64) for `ai_placeholder_mappings`. |
| `AI_PRIVACY_DEV_PLAINTEXT_STORE=1` | **Local only**: store vault “ciphertext” as plaintext (tests / dev without keys). |

## 5. Trace fields

`answer_traces.privacy_json` holds redaction summary, rehydration mode, and policy context snapshot **without** cleartext secrets.

## 6. Known limitations

- **Embeddings** and other **non-chat** OpenAI calls (e.g. `text-embedding-3-small` via `internal/llm/embed.go`) are **intentionally outside** `PrivacyGateway` for v1. Rationale and guarantees: [adr/0013-embeddings-privacy-boundary.md](./adr/0013-embeddings-privacy-boundary.md). Operators should assume **provider-side logging** may see chunk text unless they use a self-hosted or air-gapped embedding endpoint.
- **NER hook** is a no-op until a concrete `EntityExtractor` is registered.
