# ADR-0008: AI-Generated Artifacts Default to Derived

## Status

Accepted

## Context

The platform uses AI for:
- summarization
- extraction
- decision candidate generation
- insight generation
- link suggestions
- stale detection assistance
- scoped Q&A

A major trust failure would occur if AI-generated artifacts were treated as canonical by default.

Examples of risky failure:
- an extracted decision appears authoritative without confirmation
- a generated SOP draft looks like active policy
- a summary is mistaken for source truth
- synthesized content outranks stronger reviewed artifacts because its status is unclear

The product’s trust model depends on explicit distinction between:
- canonical platform truth
- mirrored external authority
- derived generated artifacts

## Decision

AI-generated reusable artifacts must default to `derived_artifact` truth mode.

They may transition to stronger states only through explicit governed workflows such as:
- review
- approval
- confirmation
- controlled publication
- canonicalization steps defined by product behavior

## Consequences

### Positive

- stronger trust semantics
- reduced authority confusion
- safer defaults for AI-generated content
- better UX clarity
- cleaner governance and publication rules

### Negative

- more review overhead for some workflows
- less immediate “magic” in demos
- additional lifecycle transitions required for some generated objects

## Guardrails

- AI-generated artifacts must display their derived status clearly
- generated outputs must preserve provenance and citations where possible
- generated outputs must receive owner, domain, sensitivity, policy source, and workflow state
- only explicit workflows may promote derived artifacts into stronger authoritative states
- retrieval and ranking should not silently treat derived artifacts as equal to approved canonical content

## Revisit triggers

Revisit if:
- a narrowly defined class of low-risk generated artifacts proves safe for automatic stronger classification
- review and validation pipelines mature enough to justify specific exceptions

The default should remain derived-first.