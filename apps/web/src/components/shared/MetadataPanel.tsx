import type { ReactNode } from "react";

export type MetadataRow = { label: string; value: ReactNode };

export function MetadataPanel({ rows }: { rows: MetadataRow[] }) {
  return (
    <dl className="divide-y divide-neutral-100 rounded-lg border border-neutral-200 bg-white">
      {rows.map((r) => (
        <div key={r.label} className="grid grid-cols-1 gap-1 px-4 py-3 sm:grid-cols-3 sm:gap-4">
          <dt className="text-xs font-medium uppercase tracking-wide text-neutral-500">{r.label}</dt>
          <dd className="text-sm text-neutral-900 sm:col-span-2">{r.value}</dd>
        </div>
      ))}
    </dl>
  );
}
