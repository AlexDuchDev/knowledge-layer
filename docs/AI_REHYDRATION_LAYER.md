# AI rehydration layer

Rehydration restores **typed placeholders** to original values **only** when policy, output context, and viewer permissions allow. See [AI_PRIVACY_POLICY.md](./AI_PRIVACY_POLICY.md) and [AI_SANITIZATION_LAYER.md](./AI_SANITIZATION_LAYER.md).

## 1. Storage

- Table `ai_placeholder_mappings` (migration `000033_ai_placeholder_vault`): encrypted `ciphertext` + `nonce`, `correlation_id` (Ask trace id or job run id), optional `principal_id` / `job_run_id`, `expires_at`.
- Encryption: AES-256-GCM via `AI_PRIVACY_VAULT_KEY` (32-byte raw or base64). Local dev may set `AI_PRIVACY_DEV_PLAINTEXT_STORE=1` (**never** in production).

## 2. Services

- `SecureEntityMapStore` — `PutBatch`, `ListDecryptedForCorrelation` (internal only).
- `RehydrationService.RehydrateFromVault` — decrypts rows for a correlation id and replaces placeholders in model output text.
- `RehydrateFromTokenizer` — same-request path using in-memory `PlaceholderTokenizer` (no DB read).

## 3. Rules (never blind rehydration)

1. **Output cap** — caller passes `RehydrationMode` (`none` / `partial` / `full`). It is clamped by effective policy per sensitive type.
2. **Per-type policy** — `EffectivePolicy.RehydrationFor(type)` must allow restoration for the requested cap (e.g. `full` requires per-type `full`).
3. **Output policy** — `auto_publish` or high `output_sensitivity` forces **`none`** in the current implementation.
4. **Full mode** — additionally requires `view` on every **evidence entity** via `AccessEvaluator` (fail closed to partial if any check fails).

## 4. Traces and APIs

- Traces store **privacy metadata** (counts, modes), not vault cleartext.
- Public API responses must not return raw mapping rows.

## 5. Limitations

- Placeholder replacement is string-based; unusual collisions in model output are possible but unlikely with `TYPE_N` tokens.
- Async multi-hop jobs should reuse the same `correlation_id` and respect TTL.

See [AI_PRIVACY_FLOW.md](./AI_PRIVACY_FLOW.md) for Ask/job wiring.
