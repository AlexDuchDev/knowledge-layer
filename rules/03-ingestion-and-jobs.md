# Ingestion and Knowledge Jobs Rules

This project treats ingestion and knowledge jobs as governed infrastructure.

## Ingestion rules

- Only ingest from explicitly configured source feeds
- Every source feed must have owner, domain, sensitivity, allowed jobs, and ingestion mode before activation
- Connector auth is not enough governance
- Connectors must stop at raw artifacts and normalized records
- Do not publish canonical objects directly from connector code
- Preserve raw artifacts for provenance and reprocessing
- Keep normalization versioned and explainable
- Deduplicate carefully without destroying source evidence

## Telegram v1 rule

Telegram is ingestion-only in v1.

Do not implement Telegram as:
- unrestricted bot interface
- universal output channel
- generic assistant surface

Only explicitly configured Telegram feeds are allowed.

## Knowledge job rules

Knowledge jobs are first-class governed objects.

Every meaningful job must define:
- purpose
- owner
- source scope
- trigger
- allowed operators
- output route
- output domain
- output sensitivity
- publication mode
- review requirement

## Job execution rules

- Jobs run only against allowed source scope
- Scheduled jobs run under governed stored scope, not broad worker privileges
- Job runs must preserve input scope snapshot and execution trace
- Job outputs must receive owner, domain, sensitivity, truth mode, provenance, workflow state, and policy source
- High-authority generated outputs require review
- When unsure, route outputs to draft or review

## Operational expectations

- Support retries where appropriate
- Preserve partial success visibility
- Keep run history inspectable
- Emit audit events for sensitive job behavior
- Prefer repeatable operational jobs over vague “smart automation”