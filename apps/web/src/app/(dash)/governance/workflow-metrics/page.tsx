"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson } from "@/lib/api";

export default function WorkflowMetricsPage() {
  const [m, setM] = useState<{ open_review_tasks: number; overdue_review_tasks: number } | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setM(await apiJson("/governance/workflow-metrics"));
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
      }
    })();
  }, []);

  return (
    <main className="mx-auto max-w-lg px-6 py-10">
      <Link href="/governance" className="text-sm text-blue-700 underline">
        Governance
      </Link>
      <h1 className="mt-4 text-2xl font-semibold">Workflow metrics</h1>
      {err ? <p className="mt-4 text-sm text-red-700">{err}</p> : null}
      {m ? (
        <ul className="mt-6 space-y-2 text-sm">
          <li>Open review tasks: {m.open_review_tasks}</li>
          <li>Overdue review tasks: {m.overdue_review_tasks}</li>
        </ul>
      ) : null}
    </main>
  );
}
