# ADR-0002: Enforce Access Before Retrieval

## Status

Accepted

## Context

The platform will support:
- search
- hybrid retrieval
- relation-aware retrieval
- AI-assisted Q&A
- AI-assisted summarization and extraction

A common failure mode in AI systems is:
- retrieve broad context first
- filter or redact later
- assume prompt instructions will prevent misuse

This is unsafe for a governed company knowledge platform because:
- disallowed content may influence ranking or generation
- hidden context may leak through summaries or phrasing
- cached broad retrieval can violate principal-specific visibility
- citations do not repair an already unsafe context assembly step

## Decision

The system will enforce access before retrieval.

For any search, retrieval, or AI request, the system must:
1. identify the principal
2. resolve allowed scope
3. retrieve only within allowed scope
4. rank only allowed results
5. assemble AI context only from allowed results

The system must never:
- retrieve broadly and filter later
- pass disallowed content to the model
- use broad cached context across principals
- treat citations as a substitute for access enforcement

## Consequences

### Positive

- safer AI behavior
- stronger trust model
- cleaner auditability
- clearer permission semantics
- reduced risk of sensitive context leakage

### Negative

- retrieval implementation is more complex
- ranking may need principal-aware query shaping
- debugging retrieval becomes more demanding
- some users may get narrower answers than broad systems would provide

## Guardrails

- access logic must be test-covered
- relation expansion must re-check access on each linked object
- background jobs must also obey governed scope
- answer traces must preserve scope and evidence set
- search indexes must not become a permission bypass path

## Revisit triggers

Revisit only if:
- access architecture changes materially
- a new retrieval substrate forces a different enforcement layer

The underlying principle should remain unchanged.