"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson } from "@/lib/api";

type Row = { query: string; count: number; avg_hits: number; weak_pattern: boolean };

export default function SearchInsightsPage() {
  const [rows, setRows] = useState<Row[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setRows(await apiJson<Row[]>("/ops/search-insights?limit=50"));
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setRows(null);
      }
    })();
  }, []);

  return (
    <main className="mx-auto max-w-4xl px-6 py-10">
      <Link href="/" className="text-sm text-blue-700 underline">
        Home
      </Link>
      <h1 className="mt-4 text-2xl font-semibold">Search insights</h1>
      <p className="mt-2 text-sm text-neutral-600">Aggregated queries (30d). Weak = low average hit count.</p>
      {err ? <p className="mt-4 text-sm text-red-700">{err}</p> : null}
      <ul className="mt-6 space-y-2 text-sm">
        {(rows ?? []).map((r) => (
          <li key={r.query} className="rounded border border-neutral-200 p-2">
            <span className="font-medium">{r.query || "(empty)"}</span>
            <span className="ml-2 text-xs text-neutral-600">
              count {r.count} · avg hits {(r.avg_hits ?? 0).toFixed(1)}
              {r.weak_pattern ? " · weak" : ""}
            </span>
          </li>
        ))}
      </ul>
    </main>
  );
}
