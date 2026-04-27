# Documentation update checklist

Use per task or PR. Pair with [DOCS_IMPACT_MAP.md](DOCS_IMPACT_MAP.md) to find concrete files.

**Policy:** [DOCS_MAINTENANCE_POLICY.md](DOCS_MAINTENANCE_POLICY.md)

---

## How to use

1. Identify which sections below apply to your change.
2. Complete the checklists for those sections.
3. Finish with the **completion report** at the bottom (required).

---

## 1. Backend contract changes

Applies when HTTP routes, request/response shapes, status codes, or error semantics change.

- [ ] [API_SURFACE_V1.md](API_SURFACE_V1.md) updated (endpoints, payloads, auth requirements).
- [ ] [ACCESS_MODEL.md](ACCESS_MODEL.md) updated if auth or public vs authenticated behavior changed.
- [ ] Related domain doc updated ([DOMAIN_MODEL.md](DOMAIN_MODEL.md), [INGESTION_API.md](INGESTION_API.md), etc.) if the contract reflects domain semantics.
- [ ] Example requests/responses in docs match actual handler behavior.

---

## 2. Product behavior changes

Applies when business rules, governance, retrieval scoping, job behavior, or connector semantics change (without necessarily changing route names).

- [ ] [DOMAIN_MODEL.md](DOMAIN_MODEL.md) if entities, relationships, or lifecycle changed.
- [ ] [ACCESS_MODEL.md](ACCESS_MODEL.md) / [permission-system.md](permission-system.md) if enforcement or grants changed.
- [ ] [INGESTION_AND_CONNECTORS.md](INGESTION_AND_CONNECTORS.md) / family docs if ingestion changed.
- [ ] [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md) / [knowledge-jobs-engine.md](knowledge-jobs-engine.md) if jobs changed.
- [ ] [AI_RETRIEVAL_GOVERNANCE.md](AI_RETRIEVAL_GOVERNANCE.md) if retrieval, ask, or AI governance changed.
- [ ] [MODULE_BOUNDARIES.md](MODULE_BOUNDARIES.md) or [ARCHITECTURE.md](ARCHITECTURE.md) if module responsibilities shifted.

---

## 3. UI changes

Applies when routes, layouts, flows, or user-visible copy change.

- [ ] [INFORMATION_ARCHITECTURE_V1.md](INFORMATION_ARCHITECTURE_V1.md) and/or [control-plane-ui-ia.md](control-plane-ui-ia.md) if navigation or IA changed.
- [ ] [USER_GUIDE_V1.md](USER_GUIDE_V1.md) and/or [ADMIN_GUIDE_V1.md](ADMIN_GUIDE_V1.md) if user/admin steps changed.
- [ ] [SEARCH_AND_QA_UX.md](SEARCH_AND_QA_UX.md) if search/ask/explorer UX changed.
- [ ] Empty states, tooltips, onboarding copy in `apps/web/src/**/*.tsx` reviewed and aligned with docs (or docs updated to match intentional copy).

---

## 4. Deployment / runtime changes

Applies when env vars, startup validation, health/ops, workers, compose, Docker, or CI change.

- [ ] [`.env.example`](../.env.example) updated if variables added/removed/renamed.
- [ ] [CONFIG_ENV.md](CONFIG_ENV.md) and/or [Config and environments.md](Config%20and%20environments.md).
- [ ] [DEPLOY_CHECKLIST.md](DEPLOY_CHECKLIST.md), [PRODUCTION_HARDENING.md](PRODUCTION_HARDENING.md), and/or [PRODUCTION_GO_LIVE_CHECKLIST.md](PRODUCTION_GO_LIVE_CHECKLIST.md) as appropriate.
- [ ] [STAGING_SMOKE_TEST.md](STAGING_SMOKE_TEST.md) if smoke steps or expectations change.
- [ ] [SELF_HOSTED.md](SELF_HOSTED.md) / [OPERATIONS.md](OPERATIONS.md) if operator procedures change.

---

## 5. Terminology / concept renames

Applies when introducing or renaming product vocabulary.

- [ ] [DOMAIN_MODEL.md](DOMAIN_MODEL.md) — canonical definitions.
- [ ] [USER_GUIDE_V1.md](USER_GUIDE_V1.md) — user-facing terms consistent.
- [ ] Specialist docs (builders, connectors, jobs) scanned for old terms.
- [ ] Consider adding or updating [GLOSSARY.md](GLOSSARY.md) if many terms moved.

---

## Completion report (required)

Paste into PR description or agent final message:

### Code files changed

- 

### Documentation files changed

- (list `docs/*`, `AGENTS.md`, `CONTRIBUTING.md`, `.env.example`, etc., or write **None**)

### In-product guidance changed (paths)

- (list `apps/web/...` or **None**)

### If no documentation changes

Explain why the change does not affect operator docs, user docs, or in-product guidance (specific reason; not “N/A” alone).

---

**Reference template:** [templates/TASK_AND_PR_DOC_IMPACT.md](templates/TASK_AND_PR_DOC_IMPACT.md)
