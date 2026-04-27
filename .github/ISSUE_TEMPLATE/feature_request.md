---
name: Feature request
about: Propose a capability that isn't covered by the current scope.
title: "[feat] "
labels: [enhancement, triage]
---

<!--
Before opening:
- Skim docs/OSS_V1_SCOPE.md and docs/LIMITATIONS.md — your idea might already be planned, deferred, or intentionally out of scope.
- For new connectors, see the family table in docs/CONNECTOR_CAPABILITY_MATRIX.md and the contract in docs/connector-framework.md.
-->

## What problem does this solve

<!-- Who is the user, what are they trying to do, why is the current product unable to support it. Concrete example > abstract description. -->

## Proposed direction

<!-- One paragraph. The minimum-viable shape; not a full design. -->

## Where would this live

Pick one (or call out cross-cutting):

- [ ] **Connector** under `apps/api/internal/ingestion_connectors/adapters/<new>/` (see the Slack adapter for the reference shape, including webhook support if push-based)
- [ ] **Knowledge job** processor under `apps/api/internal/knowledge_jobs/`
- [ ] **Retrieval / AI** layer (`internal/retrieval/`, `internal/qa/`, `internal/ai/privacy/`)
- [ ] **Governance** (`internal/governance/`, `internal/review/`)
- [ ] **Control plane UI** (`apps/web/src/app/control-plane/...`)
- [ ] **Product surface UI** (`apps/web/src/app/(dash)/...`)
- [ ] **Observability** (`internal/platform/metrics/`, worker `/ops/health`, audit events)
- [ ] **Cross-cutting** — describe below

## Governance checklist

These are the questions we ask before any feature lands. Filling them in early shortens triage.

- **Access**: who is allowed to use this? Can a denied principal see partial output? What `AccessEvaluator` action does the path use?
- **Audit**: what new event types (if any) belong in `audit_events`? Should existing dashboards surface them?
- **Privacy**: does the path send data to an LLM? If so, is it routed through `ai/privacy.PrivacyGateway`? Any new sensitive entity types?
- **Provenance**: if this generates content, what citations or source links must follow it?
- **Failure mode**: what happens when the dependency is unavailable (Redis down, OpenSearch off, Neo4j unset, LLM key missing)? Should the feature degrade or fail-closed?

## Alternatives considered

<!-- Brief: what else did you think about and why does the proposed direction win? "Just do it manually" is a valid alternative. -->

## Anything else

<!-- Mocks, links to similar features in other products, related issues. -->
