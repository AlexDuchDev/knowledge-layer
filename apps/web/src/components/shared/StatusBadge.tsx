import type {
  GovernanceLifecycleState,
  JobSurfaceKind,
  OperationalState,
  ProvenanceState,
  SourceSurfaceKind,
  VisibilityHint,
} from "@/lib/objectState";
import {
  governanceSemantics,
  jobSurfaceSemantics,
  operationalSemantics,
  provenanceSemantics,
  sourceSurfaceSemantics,
  visibilitySemantics,
} from "@/lib/objectState";

export type StatusBadgeVariant =
  | { kind: "governance"; state: GovernanceLifecycleState }
  | { kind: "operational"; state: OperationalState }
  | { kind: "provenance"; state: ProvenanceState }
  | { kind: "job_surface"; state: JobSurfaceKind }
  | { kind: "source_surface"; state: SourceSurfaceKind }
  | { kind: "visibility"; state: VisibilityHint };

export function StatusBadge({ variant }: { variant: StatusBadgeVariant }) {
  const { label, className } = (() => {
    switch (variant.kind) {
      case "governance":
        return governanceSemantics(variant.state);
      case "operational":
        return operationalSemantics(variant.state);
      case "provenance":
        return provenanceSemantics(variant.state);
      case "job_surface":
        return jobSurfaceSemantics(variant.state);
      case "source_surface":
        return sourceSurfaceSemantics(variant.state);
      case "visibility":
        return visibilitySemantics(variant.state);
    }
  })();

  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${className}`}>{label}</span>
  );
}
