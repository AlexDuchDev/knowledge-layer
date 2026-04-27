# REPO_STRUCTURE.md

## 1. Purpose

This document defines the recommended repository structure for v1.

The goal is to:
- reflect bounded domains clearly
- support frontend, backend, and worker separation
- keep shared contracts visible
- make docs first-class
- keep room for future growth without early sprawl

---

## 2. Top-level layout

```text
.
├─ apps/
│  ├─ web/
│  ├─ api/
│  └─ workers/   # README only; worker binary: api/cmd/jobworker
├─ packages/
│  └─ shared/
├─ docs/
│  ├─ adr/
│  ├─ PRODUCT.md
│  ├─ PRD-v1.md
│  ├─ ARCHITECTURE.md
│  ├─ DOMAIN_MODEL.md
│  ├─ DOMAIN_MODEL_CONTRACT.md
│  ├─ OSS_V1_SCOPE.md
│  ├─ RUNBOOK_STAGING_PROD.md
│  ├─ ACCESS_MODEL.md
│  ├─ INGESTION_AND_CONNECTORS.md
│  ├─ KNOWLEDGE_JOBS.md
│  ├─ AI_RETRIEVAL_GOVERNANCE.md
│  ├─ IMPLEMENTATION_PLAN.md
│  ├─ REPO_STRUCTURE.md
│  ├─ INITIAL_SCHEMA_OUTLINE.md
│  ├─ API_SURFACE_V1.md
│  └─ ADMIN_UI_V1.md
├─ .cursor/
│  └─ rules/
├─ AGENTS.md
├─ README.md
├─ Makefile
├─ .env.example
└─ docker-compose.yml