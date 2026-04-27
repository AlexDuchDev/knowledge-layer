"use client";

import { useCallback, useState } from "react";
import Link from "next/link";
import { apiBase, apiJson, principalUserId } from "@/lib/api";

type Json = Record<string, unknown>;

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

export default function AuditPage() {
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [rows, setRows] = useState<Json[] | null>(null);

  const run = useCallback(async (fn: () => Promise<void>) => {
    setErr(null);
    setBusy(true);
    try {
      await fn();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, []);

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Audit events</h1>
          <p className="mt-1 text-sm text-neutral-600">
            Requires identity admin. API {apiBase()} · principal {principalUserId()}
          </p>
        </div>
        <Link href="/" className="text-sm text-blue-700 underline">
          Home
        </Link>
      </div>
      {err ? <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      <button
        type="button"
        className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy}
        onClick={() =>
          run(async () => {
            setRows(await apiJson<Json[]>("/audit-events?limit=80"));
          })
        }
      >
        {busy ? "Loading…" : "GET /audit-events"}
      </button>
      {rows ? (
        <ul className="mt-4 space-y-2 font-mono text-xs">
          {rows.map((r) => (
            <li key={asStr(r.id)} className="rounded border border-neutral-100 px-2 py-1">
              {asStr(r.event_type)} · {asStr(r.target_type)} · {asStr(r.id)}
            </li>
          ))}
        </ul>
      ) : null}
    </main>
  );
}
