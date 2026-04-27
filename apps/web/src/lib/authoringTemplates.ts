import { ENTITY_TYPES } from "@/lib/entityTypes";

export type AuthoringTemplate = {
  id: string;
  label: string;
  description: string;
  type: string;
  title: string;
  summary: string;
  body: string;
};

/** Opinionated starters for governed artifacts — all default to draft + derived; authors set truth_mode explicitly when needed. */
export const AUTHORING_TEMPLATES: AuthoringTemplate[] = [
  {
    id: "sop",
    label: "SOP",
    description: "Repeatable procedure with ownership and verification.",
    type: ENTITY_TYPES.process_sop,
    title: "SOP: ",
    summary: "One line: what this procedure achieves and when to use it.",
    body: `## Purpose
## Scope / when to use
## Preconditions
## Steps
1.
2.
## Roles & responsibilities
## Verification / quality checks
## References & related decisions
`,
  },
  {
    id: "policy",
    label: "Policy",
    description: "Rule set with applicability and exceptions path.",
    type: ENTITY_TYPES.policy,
    title: "Policy: ",
    summary: "Plain-language summary of the rule and who it applies to.",
    body: `## Policy statement
## Scope (who/what/when)
## Requirements
## Exceptions & escalation
## Definitions
## Related standards
`,
  },
  {
    id: "decision",
    label: "Decision",
    description: "Decision record with context and consequences.",
    type: ENTITY_TYPES.decision,
    title: "Decision: ",
    summary: "What we decided in one sentence.",
    body: `## Context
## Decision
## Rationale
## Consequences / impact
## Alternatives considered
## Owners & review cadence
`,
  },
  {
    id: "meeting_summary",
    label: "Meeting summary",
    description: "Capture outcomes without losing traceability.",
    type: ENTITY_TYPES.meeting_summary,
    title: "Meeting: ",
    summary: "Date, attendees (roles), and primary outcome.",
    body: `## Meeting
- Date:
- Attendees (roles):

## Agenda
## Decisions
## Action items (owner · due)
## Parking lot / follow-ups
`,
  },
  {
    id: "role_handbook",
    label: "Role handbook",
    description: "Onboarding-oriented responsibilities and resources (uses process_sop type until a dedicated type exists).",
    type: ENTITY_TYPES.process_sop,
    title: "Handbook: ",
    summary: "Role name and mission in one line.",
    body: `## Role mission
## Core responsibilities
## Decision rights
## Key collaborators
## Systems & resources
## Cadence (1:1s, reviews)
## Related policies / SOPs
`,
  },
];

export function templateById(id: string): AuthoringTemplate | undefined {
  return AUTHORING_TEMPLATES.find((t) => t.id === id);
}
