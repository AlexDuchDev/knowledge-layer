import { BROWSE_ROUTES, ENTITY_TYPES } from "@/lib/entityTypes";

/** URL segment -> entities.type */
export const KNOWLEDGE_SLUG_TO_TYPE: Record<string, string> = {
  decisions: ENTITY_TYPES.decision,
  policies: ENTITY_TYPES.policy,
  processes: ENTITY_TYPES.process_sop,
  meetings: ENTITY_TYPES.meeting_summary,
  insights: ENTITY_TYPES.insight,
  projects: ENTITY_TYPES.project,
};

export function labelForSlug(slug: string): string {
  const t = KNOWLEDGE_SLUG_TO_TYPE[slug];
  if (!t) return slug;
  return BROWSE_ROUTES[t]?.label ?? slug;
}

/** Browse path for an `entities.type` value when it is in the preset map; otherwise null. */
export function browsePathForEntityType(entityType: string): { href: string; label: string } | null {
  const meta = BROWSE_ROUTES[entityType];
  return meta ? { href: meta.path, label: meta.label } : null;
}
