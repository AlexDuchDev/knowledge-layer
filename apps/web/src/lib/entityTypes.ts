/** Canonical entity `type` strings — keep in sync with `packages/shared` and Go home/search presets. */

export const ENTITY_TYPES = {
  decision: "decision",
  policy: "policy",
  process_sop: "process_sop",
  meeting_summary: "meeting_summary",
  insight: "insight",
  digest: "digest",
  project: "project",
  reference_document: "reference_document",
} as const;

export type EntityTypePreset = (typeof ENTITY_TYPES)[keyof typeof ENTITY_TYPES];

/** After governance approve, entities move to published + approved (see review Approve in API). */
export const LIFECYCLE_PUBLISHED = "published";
export const APPROVAL_APPROVED = "approved";

export const BROWSE_ROUTES: Record<string, { path: string; label: string }> = {
  [ENTITY_TYPES.decision]: { path: "/decisions", label: "Decisions" },
  [ENTITY_TYPES.policy]: { path: "/policies", label: "Policies" },
  [ENTITY_TYPES.process_sop]: { path: "/processes", label: "Processes / SOPs" },
  [ENTITY_TYPES.meeting_summary]: { path: "/meetings", label: "Meetings" },
  [ENTITY_TYPES.insight]: { path: "/insights", label: "Insights" },
  [ENTITY_TYPES.project]: { path: "/projects", label: "Projects" },
};
