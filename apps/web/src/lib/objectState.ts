/**
 * Shared labels and badge styling for governance, operational, and object-kind semantics.
 * Keep status truth in the API; this layer is presentation only.
 */

export type GovernanceLifecycleState =
  | "draft"
  | "in_review"
  | "approved"
  | "active"
  | "stale"
  | "archived"
  | "superseded";

export type OperationalState =
  | "active"
  | "inactive"
  | "failed"
  | "running"
  | "pending"
  | "completed";

/** Distinguishes catalog/template objects from live configuration rows. */
export type ProvenanceState = "preset" | "instantiated";

/** Configured job definition vs a single execution. */
export type JobSurfaceKind = "configured_job" | "job_run";

/** Connector plugin vs configured feed instance. */
export type SourceSurfaceKind = "connector" | "source_feed";

export type VisibilityHint = "standard" | "restricted" | "leadership_only";

const governanceLabels: Record<GovernanceLifecycleState, string> = {
  draft: "Draft",
  in_review: "In review",
  approved: "Approved",
  active: "Active",
  stale: "Stale",
  archived: "Archived",
  superseded: "Superseded",
};

const governanceClasses: Record<GovernanceLifecycleState, string> = {
  draft: "bg-neutral-100 text-neutral-800 ring-neutral-200",
  in_review: "bg-amber-50 text-amber-900 ring-amber-200",
  approved: "bg-emerald-50 text-emerald-900 ring-emerald-200",
  active: "bg-sky-50 text-sky-900 ring-sky-200",
  stale: "bg-orange-50 text-orange-900 ring-orange-200",
  archived: "bg-neutral-100 text-neutral-600 ring-neutral-200",
  superseded: "bg-violet-50 text-violet-900 ring-violet-200",
};

const operationalLabels: Record<OperationalState, string> = {
  active: "Active",
  inactive: "Inactive",
  failed: "Failed",
  running: "Running",
  pending: "Pending",
  completed: "Completed",
};

const operationalClasses: Record<OperationalState, string> = {
  active: "bg-emerald-50 text-emerald-900 ring-emerald-200",
  inactive: "bg-neutral-100 text-neutral-600 ring-neutral-200",
  failed: "bg-red-50 text-red-900 ring-red-200",
  running: "bg-sky-50 text-sky-900 ring-sky-200",
  pending: "bg-amber-50 text-amber-900 ring-amber-200",
  completed: "bg-neutral-100 text-neutral-800 ring-neutral-200",
};

const provenanceLabels: Record<ProvenanceState, string> = {
  preset: "Preset",
  instantiated: "Instantiated",
};

const provenanceClasses: Record<ProvenanceState, string> = {
  preset: "bg-violet-50 text-violet-900 ring-violet-200",
  instantiated: "bg-neutral-50 text-neutral-900 ring-neutral-200",
};

const jobSurfaceLabels: Record<JobSurfaceKind, string> = {
  configured_job: "Configured job",
  job_run: "Job run",
};

const jobSurfaceClasses: Record<JobSurfaceKind, string> = {
  configured_job: "bg-neutral-50 text-neutral-900 ring-neutral-200",
  job_run: "bg-sky-50 text-sky-900 ring-sky-200",
};

const sourceSurfaceLabels: Record<SourceSurfaceKind, string> = {
  connector: "Connector",
  source_feed: "Source feed",
};

const sourceSurfaceClasses: Record<SourceSurfaceKind, string> = {
  connector: "bg-neutral-100 text-neutral-800 ring-neutral-200",
  source_feed: "bg-teal-50 text-teal-900 ring-teal-200",
};

const visibilityLabels: Record<VisibilityHint, string> = {
  standard: "Standard access",
  restricted: "Restricted",
  leadership_only: "Leadership only",
};

const visibilityClasses: Record<VisibilityHint, string> = {
  standard: "bg-neutral-50 text-neutral-700 ring-neutral-200",
  restricted: "bg-amber-50 text-amber-900 ring-amber-200",
  leadership_only: "bg-neutral-900 text-white ring-neutral-700",
};

export function governanceSemantics(state: GovernanceLifecycleState): { label: string; className: string } {
  return { label: governanceLabels[state], className: governanceClasses[state] };
}

export function operationalSemantics(state: OperationalState): { label: string; className: string } {
  return { label: operationalLabels[state], className: operationalClasses[state] };
}

export function provenanceSemantics(state: ProvenanceState): { label: string; className: string } {
  return { label: provenanceLabels[state], className: provenanceClasses[state] };
}

export function jobSurfaceSemantics(kind: JobSurfaceKind): { label: string; className: string } {
  return { label: jobSurfaceLabels[kind], className: jobSurfaceClasses[kind] };
}

export function sourceSurfaceSemantics(kind: SourceSurfaceKind): { label: string; className: string } {
  return { label: sourceSurfaceLabels[kind], className: sourceSurfaceClasses[kind] };
}

export function visibilitySemantics(hint: VisibilityHint): { label: string; className: string } {
  return { label: visibilityLabels[hint], className: visibilityClasses[hint] };
}
