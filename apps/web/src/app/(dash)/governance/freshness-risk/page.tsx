"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson } from "@/lib/api";

type Row = {
  id: string;
  title: string;
  type: string;
  risk_score: number;
  freshness_status: string;
  lifecycle_state: string;
};

export default function FreshnessRiskPage() {
  const [rows, setRows] = useState<Row[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setRows(await apiJson<Row[]>("/governance/freshness-risk?limit=80"));
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setRows(null);
      }
    })();
  }, []);

  return (
    <main className="mx-auto max-w-4xl px-6 py-10">
      <Link href="/governance" className="text-sm text-blue-700 underline">
        Governance
      </Link>
      <h1 className="mt-4 text-2xl font-semibold">Freshness risk</h1>
      <p className="mt-2 text-sm text-neutral-600">Heuristic risk score: policy/SOP weight, sensitivity, freshness.</p>
      {err ? <p className="mt-4 text-sm text-red-700">{err}</p> : null}
      <ol className="mt-6 space-y-2 text-sm">
        {(rows ?? []).map((r) => (
          <li key={r.id} className="flex justify-between gap-2 rounded border p-2">
            <Link href={`/entities/${r.id}`} className="text-blue-700 underline">
              {r.title}
            </Link>
            <span className="shrink-0 text-xs text-neutral-600">
              score {r.risk_score} · {r.freshness_status}
            </span>
          </li>
        ))}
      </ol>
    </main>
  );
}
