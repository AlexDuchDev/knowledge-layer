# ADR-0010: Review Is Required for High-Authority Generated Output

## Status

Accepted

## Context

The platform generates outputs that may influence:
- operational decisions
- planning alignment
- process execution
- policy interpretation
- institutional memory

Some outputs carry higher authority risk than others.

Examples of higher-risk generated outputs:
- extracted decisions used to guide future work
- planning summaries used for execution alignment
- SOP drafts
- policy drafts
- leadership summaries
- cross-domain sensitive syntheses

If such outputs are published or treated as authoritative without review:
- errors can propagate into team behavior
- authority can be misassigned to AI outputs
- trust in the system collapses when mistakes are found
- governance becomes performative instead of real

## Decision

High-authority generated outputs require review before publication or authoritative use.

This requirement applies to outputs that:
- may shape operational behavior materially
- may be mistaken for current truth
- affect sensitive domains
- create or update governed artifacts with significant authority weight

Low-risk helper outputs may follow lighter routes if explicitly allowed by policy.

## Consequences

### Positive

- stronger trust model
- lower risk of unsafe publication
- clearer human accountability
- better quality control for important outputs
- stronger alignment with governed product positioning

### Negative

- slower path to publication for some artifacts
- review workload increases
- some users may prefer faster but riskier automation

## Guardrails

- job definitions must specify review requirement and publication mode
- AI-generated artifacts with high authority risk should default to review queues
- review surfaces must expose citations, provenance, owner, and trust state clearly
- outputs must not visually imply authority before review completes
- any exception to review requirement should be narrow, explicit, and auditable

## Revisit triggers

Revisit if:
- specific low-risk output classes prove safe for lighter handling
- validation and review tooling matures enough to support controlled exceptions

The default for high-authority generated outputs should remain review-first.