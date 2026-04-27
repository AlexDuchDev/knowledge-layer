# ADR-0001: Start with a Modular Monolith

## Status

Accepted

## Context

The platform needs to support:
- governed ingestion from multiple connectors
- canonical knowledge objects
- access control
- workflow and governance
- retrieval and AI synthesis
- auditability
- background processing

At the same time, v1 requires:
- fast iteration
- clear product learning
- limited operational overhead
- strong internal domain boundaries
- ability to evolve without early distributed systems complexity

A microservices-first approach would introduce:
- coordination overhead
- distributed data and tracing complexity
- higher operational burden
- premature ownership boundaries before product maturity

## Decision

The backend will start as a modular monolith.

This means:
- one primary deployable backend application
- explicit internal bounded domains
- separate workers/processors where useful
- no service extraction by default
- strong domain seams in code and docs

The main domain modules are:
- Identity & Access
- Knowledge Core
- Workflow & Governance
- Ingestion & Connectors
- Retrieval & Intelligence
- Platform Operations
- Knowledge Operations

## Consequences

### Positive

- lower complexity in v1
- faster product iteration
- simpler transaction boundaries
- easier end-to-end changes across related modules
- easier observability early on
- cleaner foundation before service extraction

### Negative

- some modules may become large if boundaries are not enforced
- scaling patterns are less independently tunable at first
- discipline is required to avoid “big ball of mud” drift

## Guardrails

To make this decision work:
- domain boundaries must remain explicit
- internal module APIs should be clear
- async workers should handle long-running workloads
- ADR review is required before extracting services
- code should not assume “single module = no architecture needed”

## Revisit triggers

Revisit this decision if:
- one module has materially different scaling needs
- operational isolation becomes necessary
- deployment cadence diverges sharply by module
- team ownership pressure cannot be solved within the monolith