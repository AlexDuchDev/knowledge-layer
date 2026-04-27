# Documentation Update Rules

This repo uses docs as operating context for both humans and agents.

## Core rule

If behavior, architecture, scope, or governance changes, update the relevant docs in the same work stream.

Do not leave docs behind.

## Which doc to update

- product scope, ICP, goals, non-goals -> `docs/PRODUCT.md`
- v1 requirements, UX flows, launch criteria -> `docs/PRD-v1.md`
- module boundaries, data flow, system shape -> `docs/ARCHITECTURE.md`
- entities, relations, provenance, versioning, truth modes -> `docs/DOMAIN_MODEL.md`
- permissions, inheritance, sensitivity, AI scope rules -> `docs/ACCESS_MODEL.md`
- connectors, source feeds, sync, parsing, dedup, reprocessing -> `docs/INGESTION_AND_CONNECTORS.md`
- jobs, triggers, outputs, review and routing -> `docs/KNOWLEDGE_JOBS.md`
- retrieval, citations, trust indicators, AI boundaries -> `docs/AI_RETRIEVAL_GOVERNANCE.md`
- durable architectural decisions -> `docs/adr/`

## ADR rule

Create or update an ADR when changing:
- top-level architecture shape
- service extraction decisions
- access-before-retrieval behavior
- truth classification model
- connector boundary rules
- publication or review semantics
- important data-store decisions

## Writing style for docs

- Keep docs opinionated and explicit
- Prefer short sections over vague prose
- State constraints clearly
- Preserve terminology consistency
- Do not duplicate large sections across many files unless needed
- Update the source-of-truth doc instead of adding contradictory notes elsewhere

## Agent behavior

Before implementing non-trivial work:
1. read the relevant docs
2. check whether the change conflicts with existing ADRs
3. update docs if semantics change
4. mention doc updates in the work summary

A feature is not done if the docs are now misleading.