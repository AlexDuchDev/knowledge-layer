"use client";

import { useState } from "react";
import type { EntityDetailResponse } from "@/components/EntityDetailView";

/** Internal share helper: link + compact trust context (permission-safe — same user session). */
export function ShareTrustCard({ detail }: { detail: EntityDetailResponse }) {
  const [open, setOpen] = useState(false);
  const url = typeof window !== "undefined" ? `${window.location.origin}/entities/${detail.entity.id}` : `/entities/${detail.entity.id}`;

  const card = [
    `Knowledge Layer — ${detail.entity.title || "Entity"}`,
    `URL: ${url}`,
    `Type: ${detail.entity.type}`,
    `Truth: ${detail.truth_mode} · Lifecycle: ${detail.lifecycle_state} · Approval: ${detail.approval_status}`,
    `Freshness: ${detail.freshness_status} · Canonical: ${detail.canonical_status}`,
  ].join("\n");

  return (
    <section className="mt-4 rounded-lg border border-neutral-200 bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-medium text-neutral-900">Share internally</h2>
        <button type="button" className="text-xs text-blue-800 underline" onClick={() => setOpen((o) => !o)}>
          {open ? "Hide preview" : "Copy link & trust card"}
        </button>
      </div>
      <p className="mt-1 text-xs text-neutral-600">For teammates who already have access. Recipients still go through normal permission checks.</p>
      {open ? (
        <div className="mt-3 space-y-2">
          <button
            type="button"
            className="rounded border border-neutral-300 bg-neutral-50 px-2 py-1 text-xs"
            onClick={() => void navigator.clipboard?.writeText(url)}
          >
            Copy link only
          </button>
          <button
            type="button"
            className="ml-2 rounded border border-neutral-300 bg-neutral-50 px-2 py-1 text-xs"
            onClick={() => void navigator.clipboard?.writeText(card)}
          >
            Copy trust card
          </button>
          <pre className="mt-2 max-h-40 overflow-auto rounded bg-neutral-50 p-2 text-[10px] text-neutral-800">{card}</pre>
        </div>
      ) : null}
    </section>
  );
}
