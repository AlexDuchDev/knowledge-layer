"use client";

import Link from "next/link";
import { useCallback, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { WorkflowNextSteps } from "@/components/WorkflowNextSteps";
import { apiJson, principalUserId } from "@/lib/api";

type Row = Record<string, unknown>;

export default function ApprovalsPage() {
  const [rows, setRows] = useState<Row[] | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setBusy(true);
    setErr(null);
    try {
      setRows(await apiJson<Row[]>("/governance/approval-queue"));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setRows(null);
    } finally {
      setBusy(false);
    }
  }, []);

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Approvals" }]} />
      <div className="mt-3 rounded-md border border-neutral-200 bg-white px-3 py-2">
        <WorkflowNextSteps />
      </div>
      <h1 className="text-2xl font-semibold tracking-tight">Approvals</h1>
      <p className="mt-1 text-sm text-neutral-600">
        Queue from <code className="rounded bg-neutral-100 px-1">GET /governance/approval-queue</code> (requires publish capability in granted
        domains). Principal <code className="rounded bg-neutral-100 px-1">{principalUserId().slice(0, 8)}…</code>
      </p>
      <button
        type="button"
        disabled={busy}
        className="mt-4 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        onClick={() => void load()}
      >
        {busy ? "Loading…" : "Load queue"}
      </button>
      {err ? <div className="my-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      <ul className="mt-6 space-y-2 text-sm">
        {(rows ?? []).map((r, i) => (
          <li key={i} className="rounded border border-neutral-200 bg-white px-3 py-2">
            <pre className="overflow-x-auto text-xs text-neutral-800">{JSON.stringify(r, null, 2)}</pre>
          </li>
        ))}
      </ul>
      <p className="mt-8 text-sm">
        <Link href="/governance" className="text-blue-700 underline">
          Full Governance Center
        </Link>
      </p>
    </main>
  );
}
