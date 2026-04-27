/** Shared API contract version for web ↔ API alignment. */
export const API_VERSION = "v1" as const;

export {
  ENTITY_TYPES,
  LIFECYCLE_PUBLISHED,
  APPROVAL_APPROVED,
} from "./entityTypes";

export type TruthMode = "canonical" | "mirrored" | "derived";

export type AccessDecisionResult = "allow" | "deny";
