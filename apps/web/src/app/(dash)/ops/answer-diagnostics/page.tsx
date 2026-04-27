"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson } from "@/lib/api";

type Row = {
  id: string;
  entity_id: string;
  question: string;
  answer_preview: string;
  citation_count: number;
  weak_evidence: boolean;
  model: string;
  created_at: string;
};

export default function AnswerDiagnosticsPage() {
  const [rows, setRows] = useState<Row[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setRows(await apiJson<Row[]>("/ops/answer-diagnostics?limit=80"));
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setRows(null);
      }
    })();
  }, []);

  return (
    <main className="mx-auto max-w-6xl px-6 py-10">
      <Link href="/" className="text-sm text-blue-700 underline">
        Home
      </Link>
      <h1 className="mt-4 text-2xl font-semibold">Answer diagnostics</h1>
      {err ? <p className="mt-4 text-sm text-red-700">{err}</p> : null}
      <div className="mt-6 overflow-x-auto rounded border">
        <table className="min-w-full text-left text-xs">
          <thead className="bg-neutral-50">
            <tr>
              <th className="px-2 py-1">weak</th>
              <th className="px-2 py-1">cites</th>
              <th className="px-2 py-1">question</th>
              <th className="px-2 py-1">trace</th>
            </tr>
          </thead>
          <tbody>
            {(rows ?? []).map((r) => (
              <tr key={r.id} className="border-t">
                <td className="px-2 py-1">{r.weak_evidence ? "yes" : ""}</td>
                <td className="px-2 py-1">{r.citation_count}</td>
                <td className="px-2 py-1">{r.question}</td>
                <td className="px-2 py-1">
                  <Link href={`/entities/${r.entity_id}`} className="text-blue-700 underline">
                    entity
                  </Link>{" "}
                  ·{" "}
                  <span className="font-mono">
                    {r.id.slice(0, 8)}… (use GET /answer-traces/:id)
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </main>
  );
}
