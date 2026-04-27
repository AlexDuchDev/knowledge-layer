/** Canonical entity type strings — mirror apps/web/src/lib/entityTypes.ts */

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

export const LIFECYCLE_PUBLISHED = "published";
export const APPROVAL_APPROVED = "approved";
