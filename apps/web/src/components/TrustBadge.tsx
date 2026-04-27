"use client";

export function formatTruthMode(m: string): string {
  if (m === "canonical_in_platform") return "Canonical (platform)";
  if (m === "mirrored_authority") return "Mirrored (authority)";
  if (m === "derived") return "Derived";
  return m;
}

export function TrustBadge({ truthMode }: { truthMode: string }) {
  const label = formatTruthMode(truthMode);
  const cls =
    truthMode === "canonical_in_platform"
      ? "border-emerald-200 bg-emerald-50 text-emerald-900"
      : truthMode === "mirrored_authority"
        ? "border-blue-200 bg-blue-50 text-blue-900"
        : "border-neutral-200 bg-neutral-50 text-neutral-900";
  return <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs ${cls}`}>{label}</span>;
}

/** @deprecated Prefer `TrustBadge` — alias kept for incremental refactors. */
export const TruthBadge = TrustBadge;

export type TrustLineProps = {
  truthMode: string;
  lifecycleState?: string;
  freshnessStatus?: string;
  className?: string;
};

/** Single-line trust summary for tables and compact cards. */
export function TrustLine({ truthMode, lifecycleState, freshnessStatus, className = "" }: TrustLineProps) {
  const life = lifecycleState?.trim() ? lifecycleState : "—";
  const fresh = freshnessStatus?.trim() ? freshnessStatus : "—";
  return (
    <div className={`text-xs text-neutral-700 ${className}`}>
      <span className="font-medium text-neutral-900">{formatTruthMode(truthMode)}</span>
      <span className="text-neutral-500"> · </span>
      <span>lifecycle {life}</span>
      <span className="text-neutral-500"> · </span>
      <span>freshness {fresh}</span>
    </div>
  );
}
