"use client";

import { useCallback, useId, useState } from "react";
import { formatTruthMode } from "@/components/TrustBadge";

export type TrustExplanationInput = {
  truthMode: string;
  lifecycleState: string;
  freshnessStatus: string;
  approvalStatus?: string;
  canonicalStatus?: string;
  ownerId?: string | null;
  domainId?: string;
  entityType?: string;
};

export function TrustExplanationDrawer({ meta, label = "Why trust this?" }: { meta: TrustExplanationInput; label?: string }) {
  const panelId = useId();
  const [open, setOpen] = useState(false);
  const close = useCallback(() => setOpen(false), []);

  return (
    <div className="relative inline-block">
      <button
        type="button"
        className="rounded border border-neutral-300 bg-white px-2 py-0.5 text-xs text-neutral-800 hover:bg-neutral-50"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => setOpen((v) => !v)}
      >
        {label}
      </button>
      {open ? (
        <>
          <button type="button" className="fixed inset-0 z-40 cursor-default bg-black/20" aria-label="Close" onClick={close} />
          <div
            id={panelId}
            role="dialog"
            aria-modal="true"
            className="absolute right-0 top-full z-50 mt-1 w-[min(100vw-2rem,22rem)] rounded-lg border border-neutral-200 bg-white p-3 text-xs shadow-lg"
          >
            <div className="font-medium text-neutral-900">Trust signals</div>
            <p className="mt-1 text-neutral-600">Grounded in metadata from your governed knowledge layer (not a confidence score).</p>
            <ul className="mt-2 space-y-1.5 text-neutral-800">
              <li>
                <span className="text-neutral-500">Truth mode:</span> {formatTruthMode(meta.truthMode)}
              </li>
              <li>
                <span className="text-neutral-500">Lifecycle:</span> {meta.lifecycleState}
              </li>
              <li>
                <span className="text-neutral-500">Freshness:</span> {meta.freshnessStatus}
              </li>
              {meta.approvalStatus ? (
                <li>
                  <span className="text-neutral-500">Approval:</span> {meta.approvalStatus}
                </li>
              ) : null}
              {meta.canonicalStatus ? (
                <li>
                  <span className="text-neutral-500">Canonical status:</span> {meta.canonicalStatus}
                </li>
              ) : null}
              {meta.entityType ? (
                <li>
                  <span className="text-neutral-500">Entity type:</span> {meta.entityType}
                </li>
              ) : null}
              {meta.ownerId ? (
                <li>
                  <span className="text-neutral-500">Owner:</span> <code className="rounded bg-neutral-100 px-1">{meta.ownerId}</code>
                </li>
              ) : null}
              {meta.domainId ? (
                <li>
                  <span className="text-neutral-500">Domain:</span> <code className="rounded bg-neutral-100 px-1">{meta.domainId}</code>
                </li>
              ) : null}
            </ul>
            <button type="button" className="mt-2 text-blue-700 underline" onClick={close}>
              Close
            </button>
          </div>
        </>
      ) : null}
    </div>
  );
}
