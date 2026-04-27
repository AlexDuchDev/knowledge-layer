import { ENTITY_TYPES, LIFECYCLE_PUBLISHED } from "@/lib/entityTypes";

export type SearchPresetId =
  | "all"
  | "decisions"
  | "policies"
  | "sops"
  | "meetings"
  | "insights"
  | "my_domain"
  | "approved_only";

export type PresetQuery = {
  type?: string;
  domain_id?: string;
  lifecycle_state?: string;
  approval_status?: string;
  truth_mode?: string;
};

export type SearchPreset = {
  id: SearchPresetId;
  label: string;
  description: string;
  /** Params merged into search; empty string values omitted at request time */
  params: PresetQuery;
};

export const SEARCH_PRESETS: SearchPreset[] = [
  { id: "all", label: "All types", description: "No entity type filter; still scoped to your domains.", params: {} },
  {
    id: "decisions",
    label: "Decisions",
    description: "Entity type: decision",
    params: { type: ENTITY_TYPES.decision },
  },
  {
    id: "policies",
    label: "Policies",
    description: "Entity type: policy",
    params: { type: ENTITY_TYPES.policy },
  },
  {
    id: "sops",
    label: "SOPs",
    description: "Processes and SOPs",
    params: { type: ENTITY_TYPES.process_sop },
  },
  {
    id: "meetings",
    label: "Meetings",
    description: "Meeting summaries",
    params: { type: ENTITY_TYPES.meeting_summary },
  },
  {
    id: "insights",
    label: "Insights",
    description: "Insights and derived summaries",
    params: { type: ENTITY_TYPES.insight },
  },
  {
    id: "approved_only",
    label: "Published",
    description: "Lifecycle published (aligned with post-approval publish gate)",
    params: { lifecycle_state: LIFECYCLE_PUBLISHED },
  },
];

export function presetById(id: string | null | undefined): SearchPreset | undefined {
  if (!id) return undefined;
  return SEARCH_PRESETS.find((p) => p.id === id);
}
