"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { apiJson } from "@/lib/api";

type Row = {
  id: string;
  title: string;
  type: string;
  lifecycle_state: string;
  approval_status: string;
  truth_mode: string;
  freshness_status: string;
};

export default function EditorialQueuePage() {
  const [rows, setRows] = useState<Row[] | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [hubId, setHubId] = useState("");

  const load = useCallback(async () => {
    setErr(null);
    try {
      setRows(await apiJson<Row[]>("/governance/publishing-queue?limit=50"));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setRows(null);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const hold = async (entityId: string) => {
    await apiJson("/governance/editorial/hold", {
      method: "POST",
      body: JSON.stringify({ entity_id: entityId, reason: "editorial hold" }),
    });
    void load();
  };

  const feature = async (entityId: string) => {
    if (!hubId.trim()) {
      setErr("Set hub UUID to feature into.");
      return;
    }
    await apiJson("/governance/editorial/feature", {
      method: "POST",
      body: JSON.stringify({ hub_id: hubId.trim(), entity_id: entityId, role: "featured" }),
    });
    void load();
  };

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <Link href="/governance" className="text-sm text-blue-700 underline">
        Governance
      </Link>
      <h1 className="mt-4 text-2xl font-semibold">Publishing queue</h1>
      <p className="mt-2 text-sm text-neutral-600">Published entities not on editorial hold. Requires publish-capable principal.</p>
      <div className="mt-4 flex flex-wrap gap-2">
        <input
          className="rounded border px-2 py-1 font-mono text-xs"
          placeholder="hub id for Feature action"
          value={hubId}
          onChange={(e) => setHubId(e.target.value)}
        />
        <button type="button" className="rounded bg-neutral-900 px-3 py-1 text-sm text-white" onClick={() => void load()}>
          Refresh
        </button>
      </div>
      {err ? <p className="mt-4 text-sm text-red-700">{err}</p> : null}
      <ul className="mt-6 space-y-2 text-sm">
        {(rows ?? []).map((r) => (
          <li key={r.id} className="flex flex-wrap items-center justify-between gap-2 rounded border border-neutral-200 p-3">
            <div>
              <Link href={`/entities/${r.id}`} className="font-medium text-blue-800 underline">
                {r.title}
              </Link>
              <div className="text-xs text-neutral-600">
                {r.type} · {r.truth_mode} · {r.freshness_status}
              </div>
            </div>
            <div className="flex gap-2">
              <button type="button" className="rounded border px-2 py-1 text-xs" onClick={() => void feature(r.id)}>
                Feature
              </button>
              <button type="button" className="rounded border px-2 py-1 text-xs" onClick={() => void hold(r.id)}>
                Hold
              </button>
            </div>
          </li>
        ))}
      </ul>
    </main>
  );
}
