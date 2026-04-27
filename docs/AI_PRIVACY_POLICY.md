# AI privacy policy

This document describes the **sensitive data taxonomy**, **policy model**, **scope resolution**, and **actions** used before any LLM completion call in the Knowledge Layer API.

Related: [AI_SANITIZATION_LAYER.md](./AI_SANITIZATION_LAYER.md), [AI_REHYDRATION_LAYER.md](./AI_REHYDRATION_LAYER.md), [AI_PRIVACY_FLOW.md](./AI_PRIVACY_FLOW.md).

## 1. Taxonomy

Sensitive entity types are explicit string IDs (see `apps/api/internal/ai/privacy/taxonomy.go`). They are grouped for product communication:

| Group | Meaning | Examples (entity types) |
|-------|---------|-------------------------|
| A | Personal identifiers | `person_name`, `email`, `phone`, `address`, `government_id` |
| B | Secrets | `security_secret` |
| C | Financial | `invoice_ref`, `payment_ref`, `transaction_id`, `financial_account` |
| D | Legal / contractual | `contract_ref`, `legal_ref` |
| E | Business-sensitive | `company_name`, `customer_id`, `account_id`, `internal_codename`, … |
| F | Context-specific | `hr_performance`, `custom_pattern`, … |

Policies refer to these IDs only — not informal labels.

## 2. Policy storage

Rules live in Postgres: `ai_privacy_policy_rules` (see migration `000032_ai_privacy_policy`).

Columns:

- `scope_kind`: `global` | `domain` | `source_feed` | `scenario` | `job_type` | `output_type`
- `scope_id`: nullable for `global`; otherwise UUID string or string code (e.g. scenario name, job type)
- `entity_type`: sensitive type id
- `action`: `keep` | `tokenize` | `remove` | `disallow_ai`
- `rehydration_mode`: `none` | `partial` | `full`
- `priority`: tie-breaker within the same scope tier
- `enabled`

## 3. Scope resolution

For a given invocation, `PolicyContext` carries optional `domain_id`, `source_feed_id`, `scenario`, `job_type`, `output_type`.

**Matching:** a rule applies if its scope matches the context (case-insensitive string compare for ids).

**Specificity tiers** (higher wins):

1. `output_type`
2. `job_type`
3. `scenario`
4. `source_feed`
5. `domain`
6. `global`

For each `entity_type`, among all **matching** rules, the winner is the highest tier; if tied, higher `priority` wins.

**Effective rehydration cap:** the invocation’s `EffectiveRehydration` is the minimum (most restrictive) of `rehydration_mode` across all winning rules that matched. If no rules matched, defaults to `partial` for backward-compatible behavior.

## 4. Actions

| Action | Meaning |
|--------|---------|
| `keep` | Send value to the model as-is (use sparingly). |
| `tokenize` | Replace with typed placeholder; store mapping in vault (see rehydration doc). |
| `remove` | Omit or redact value from model input. |
| `disallow_ai` | If this type is **detected** in content slated for the model, the invocation **must not** call the LLM (fail closed). |

`disallow_ai` is **per entity type**, not a global kill switch: a global rule on `security_secret` does not block Ask unless a secret is actually detected.

## 5. Default seed

The migration seeds conservative global rules (e.g. `security_secret` → `disallow_ai`, `email`/`phone` → `tokenize`). Operators can add narrower rules for domains or scenarios.

## 6. Operational notes

- Avoid turning this into a full DSL: prefer additive rules and clear entity types.
- Changes to defaults should go through migration or controlled admin tooling (future).
- For architecture alignment, see [AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md).
