# Documentation maintenance policy

This repository treats documentation as part of the **product contract**. Humans and agents (including Cursor) must keep docs, examples, and in-product guidance aligned with code.

**Related:** [DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md) (where to edit), [DOCS_UPDATE_CHECKLIST.md](DOCS_UPDATE_CHECKLIST.md) (task checklist), [templates/TASK_AND_PR_DOC_IMPACT.md](templates/TASK_AND_PR_DOC_IMPACT.md) (report template).

---

## Purpose

- Reduce silent drift between behavior and written guidance.
- Make “what to update when” explicit and path-oriented.
- Ensure deploy/runtime and security assumptions stay operator-visible.

---

## Mandatory rules

### Rule 1: Docs follow code

Any change that affects **architecture**, **API contracts**, **flows**, **setup**, **UI behavior**, or **runtime** (startup, env, health, workers) must trigger a **doc review** and updates where the impact map applies.

### Rule 2: Glossary follows concepts

If you introduce or rename a **domain concept** (entity type, grant, job, connector family, governance state, etc.), update canonical terminology in:

- [DOMAIN_MODEL.md](DOMAIN_MODEL.md) / [DOMAIN_MODEL_CONTRACT.md](DOMAIN_MODEL_CONTRACT.md) and/or [USER_GUIDE_V1.md](USER_GUIDE_V1.md), and
- any specialist doc already defining that term.

If the term surface grows, consider adding [GLOSSARY.md](GLOSSARY.md) (optional until needed).

### Rule 3: UI guidance follows behavior

If a **user-facing flow** changes (routes, steps, outcomes, errors), review and update relevant:

- empty states and onboarding copy,
- tooltips, help text, preview/description strings,
- setup wizards and control-plane explanations.

Primary code search space: `apps/web/src/**/*.tsx` (and shared copy modules if present).

### Rule 4: Deploy docs follow runtime

If **deployment**, **environment variables**, **auth/session**, **health/ops endpoints**, **workers**, **compose**, **CI**, or **infra assumptions** change, update the deployment and hardening docs listed in [DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md) (deploy/runtime section) and [`.env.example`](../.env.example) when vars change.

### Rule 5: No silent drift

A task is **not complete** if code changed **materially** and documentation stayed stale **without** an explicit justification (see **Reporting expectations** below).

**Not acceptable:**

- “Will document later” with no follow-up in the same change set.
- Updating only code comments while user-facing or operator-facing behavior changed.

**Acceptable skip:**

- Explicit **“No documentation changes”** note explaining why (e.g. pure refactor with identical behavior, typo in non-user string, test-only change). The justification must be **specific**, not generic.

---

## When docs must be updated (triggers)

Use [DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md) to map these triggers to files:

| Trigger | Typical doc touchpoints |
|---------|-------------------------|
| HTTP routes, DTOs, status codes, errors | [API_SURFACE_V1.md](API_SURFACE_V1.md), domain docs, ACCESS_MODEL if auth changes |
| Auth, permissions, grants, principals | [ACCESS_MODEL.md](ACCESS_MODEL.md), [permission-system.md](permission-system.md), [permission-flow.md](permission-flow.md) |
| Migrations / schema semantics | [DOMAIN_MODEL.md](DOMAIN_MODEL.md), [DOMAIN_MODEL_CONTRACT.md](DOMAIN_MODEL_CONTRACT.md) (long-form taxonomy), migration notes, readiness/ops docs if operator-visible |
| Connectors / ingestion | [INGESTION_AND_CONNECTORS.md](INGESTION_AND_CONNECTORS.md), [INGESTION_API.md](INGESTION_API.md), family specs, [SOURCE_FEED_SETUP_FLOW.md](SOURCE_FEED_SETUP_FLOW.md) |
| Jobs / orchestration / workers | [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md), [knowledge-jobs-engine.md](knowledge-jobs-engine.md), [job-builder.md](job-builder.md) |
| Retrieval / ask / AI governance | [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md), [SEARCH_AND_QA_UX.md](SEARCH_AND_QA_UX.md), [retrieval-ai-foundation.md](retrieval-ai-foundation.md) |
| UI routes, IA, product surface | [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md), [control-plane-ui-ia.md](control-plane-ui-ia.md), [USER_GUIDE_V1.md](USER_GUIDE_V1.md), [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md) |
| Env, config, compose, CI, hardening | [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md), [CONFIG_ENV.md](CONFIG_ENV.md), [SELF_HOSTED.md](SELF_HOSTED.md), `.env.example` |
| Durable architecture decisions | New or updated ADR under [adr/](adr/) |

---

## What counts as done

For non-trivial work:

1. Code and tests (where applicable) match intent.
2. Docs updated per impact map **or** explicit no-doc justification.
3. In-product strings reviewed when user-visible behavior changed.
4. **Task completion report** delivered (see below).

---

## Reporting expectations (every task / PR)

End with this structure (copy [templates/TASK_AND_PR_DOC_IMPACT.md](templates/TASK_AND_PR_DOC_IMPACT.md)):

1. **Code files changed** — list paths.
2. **Docs changed** — list paths under `docs/` (and root `AGENTS.md`, `CONTRIBUTING.md`, `.env.example` if applicable).
3. **In-product guidance changed** — list component/page paths under `apps/web/` (or “None”).
4. **If no doc changes** — one short paragraph: why behavior and operator/user guidance are unchanged.

This applies to **Cursor agents** and **human contributors** equally.

---

## Relationship to AGENTS.md

[AGENTS.md](../AGENTS.md) states high-level **which domain areas** map to which canonical docs. This policy adds **procedure**, **UI guidance**, **deploy/runtime**, and **mandatory reporting**. Both apply.

---

## Future automation (optional)

Possible later improvements (not required today):

- CI heuristic: fail if certain paths (e.g. `apps/api/internal/httpserver/`) change without any `docs/` diff (noisy; tune carefully).
- Script: `git diff --name-only` against [DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md) to suggest doc files for reviewers.
- Scheduled doc-audit issue or checklist in release process.

Until automation exists, **human and agent discipline** plus the checklist is the source of truth.
