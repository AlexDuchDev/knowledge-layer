# Architecture Rules

This project is a governed organizational memory platform with a modular monolith architecture.

## Always preserve these boundaries

Top-level backend domains:
- Identity & Access
- Knowledge Core
- Workflow & Governance
- Ingestion & Connectors
- Retrieval & Intelligence
- Platform Operations
- Knowledge Operations

Do not blur these domains for convenience.

## Required architectural invariants

- Access control happens before retrieval
- AI is not an authority layer
- Ingestion is not publication
- Connectors stop at raw artifacts and normalized records
- Raw artifacts are preserved
- Knowledge objects matter more than files
- Review / approval / lifecycle are native product behavior
- Policy inheritance is the default
- Fail closed when access or trust state is unclear

## Modular monolith guidance

- Prefer internal modules over new services
- Do not introduce microservices without a real bottleneck
- Keep domain APIs explicit
- Keep side effects visible
- Use workers for long-running ingestion, indexing, and job execution
- Avoid hidden cross-module coupling

## Data shape guidance

Primary truth belongs in:
- PostgreSQL transactional state
- raw artifacts in object storage
- explicit provenance, workflow, and versioning records

Derived systems such as:
- search indexes
- embeddings
- chunks

must remain rebuildable and must not become the only source of truth.

## When working on features

Before implementing:
1. identify affected domain modules
2. keep the change in the smallest correct boundary
3. check whether docs or ADRs need updates

If a change affects architecture materially, add or update an ADR.