"use client";

import Link from "next/link";
import { TrustBadge, TrustLine } from "@/components/TrustBadge";

export type BrowseEntityRow = {
  id: string;
  type: string;
  title: string;
  truth_mode: string;
  lifecycle_state: string;
  freshness_status: string;
  updated_at: string;
};

export function EntityBrowseTable({
  rows,
  loading,
  emptyMessage,
}: {
  rows: BrowseEntityRow[] | null;
  loading: boolean;
  emptyMessage: string;
}) {
  if (loading) {
    return <p className="text-sm text-neutral-600">Loading…</p>;
  }
  if (!rows || rows.length === 0) {
    return <p className="text-sm text-neutral-600">{emptyMessage}</p>;
  }
  return (
    <div className="overflow-x-auto rounded-lg border border-neutral-200">
      <table className="min-w-full text-left text-sm">
        <thead className="bg-neutral-50 text-xs font-medium text-neutral-700">
          <tr>
            <th className="px-3 py-2">Title</th>
            <th className="px-3 py-2">Type</th>
            <th className="px-3 py-2">Trust</th>
            <th className="px-3 py-2">Updated</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((e) => (
            <tr key={e.id} className="border-t border-neutral-100">
              <td className="px-3 py-2">
                <Link href={`/entities/${e.id}`} className="font-medium text-blue-800 underline">
                  {e.title || "Untitled"}
                </Link>
              </td>
              <td className="px-3 py-2 text-neutral-600">{e.type}</td>
              <td className="px-3 py-2">
                <div className="flex flex-col gap-1">
                  <TrustBadge truthMode={e.truth_mode} />
                  <TrustLine truthMode={e.truth_mode} lifecycleState={e.lifecycle_state} freshnessStatus={e.freshness_status} />
                </div>
              </td>
              <td className="whitespace-nowrap px-3 py-2 text-xs text-neutral-600">{new Date(e.updated_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
