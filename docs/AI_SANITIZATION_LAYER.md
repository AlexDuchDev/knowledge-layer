# AI sanitization layer

Detection and tokenization run **before** any LLM completion. This complements [AI_PRIVACY_POLICY.md](./AI_PRIVACY_POLICY.md).

## 1. Pipeline order (mandatory)

1. **Structured field hints** — Known fields (`title`, `external_ref`, `question`, `summary`, `body`, `chunk.text`) map to default sensitive classes where applicable (see `structured_fields.go`).
2. **Pattern detection** — Deterministic extractors (email, phone, common token shapes, contract/invoice heuristics, UUID-like, URLs, IBAN-shaped strings).
3. **Optional NER / heuristics** — `EntityExtractor` hook; default is no-op. Lowest priority vs patterns.
4. **Policy actions** — Per span: `keep`, `tokenize`, `remove`, `disallow_ai` (see policy doc).

Structured gaps are filled **around** high-confidence pattern hits so patterns **override** local structured classification (e.g. email inside a title string).

## 2. Placeholder tokenizer

- Format: `EMAIL_1`, `COMPANY_2`, `SECRET_1`, …
- **Stable within one correlation id**: the same raw value maps to the same placeholder for the whole Ask/job run.
- Implementation: `PlaceholderTokenizer` in `apps/api/internal/ai/privacy/tokenizer.go`.

## 3. Redaction report

`RedactionReport` counts spans by `entity_type`, placeholders emitted, removes, and `disallow_ai` triggers. It is safe to persist in traces (no raw values).

## 4. Code entry points

- `SanitizationService.SanitizeSegments` — batch of `TextSegment` values (see `qa/privacy_segments.go` for Ask evidence assembly).
- `NoopExtractor` — default NER hook.

## 5. Limitations

- Pattern false positives/negatives are possible; tune regexes and add structured field mappings as connectors mature.
- Embeddings / non-chat model calls are out of scope for this layer unless explicitly wired.

See [AI_PRIVACY_FLOW.md](./AI_PRIVACY_FLOW.md) for Ask/job integration.
