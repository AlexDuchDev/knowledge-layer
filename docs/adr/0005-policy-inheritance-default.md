# ADR-0005: Use Policy Inheritance as the Default Access Strategy

## Status

Accepted

## Context

The platform will manage many governed artifacts across:
- domains
- source feeds
- entity types
- job outputs
- workflow states

A naive per-object manual ACL approach would:
- create major admin overhead
- reduce trust in policy consistency
- increase configuration mistakes
- make rollout fragile
- become operationally unmanageable as content scales

At the same time, the system still needs flexibility for exceptions.

## Decision

Policy inheritance will be the default access strategy.

Most objects should inherit access behavior from:
- domain default policy
- source feed policy
- entity type policy
- job output policy

Manual object-level overrides are allowed, but they are exceptions.

Suggested precedence:
1. global deny rules
2. explicit object-level deny
3. explicit object-level allow
4. job output policy
5. source feed policy
6. entity type policy
7. domain default policy
8. role-derived defaults

## Consequences

### Positive

- lower admin burden
- more predictable policy behavior
- easier explainability
- cleaner source-to-output governance
- easier scaling of content and jobs

### Negative

- inheritance resolution logic becomes important and must be robust
- admins need good explainability tooling
- some unusual cases require override design

## Guardrails

- every object should track its policy source
- overrides must include reason and audit trail
- too many overrides should be treated as a smell
- inheritance resolution must be test-covered
- UI should explain where effective access came from

## Revisit triggers

Revisit if:
- policy inheritance becomes too rigid for real-world usage
- repeated exception patterns reveal a missing policy layer
- multi-domain publication patterns require refined precedence rules

Do not switch to object-level manual ACLs as the normal model.