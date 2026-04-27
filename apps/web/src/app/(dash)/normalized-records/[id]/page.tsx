"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { apiBase, apiJson, principalUserId } from "@/lib/api";

type NormalizedRecord = {
  id: string;
  raw_artifact_id: string;
  source_feed_id: string;
  domain_id: string;
  feed_sensitivity_level: number;
  record_type: string;
  structured_payload_json: unknown;
  record_hash: string;
  source_timestamp?: string | null;
  detected_author_ref?: string | null;
  normalization_version: number;
  created_at: string;
};

export default function NormalizedRecordDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const [err, setErr] = useState<string | null>(null);
  const [rec, setRec] = useState<NormalizedRecord | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setErr(null);
    setBusy(true);
    try {
      if (!id) throw new Error("missing id");
      const out = await apiJson<NormalizedRecord>(`/normalized-records/${id}`);
      setRec(out);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setRec(null);
    } finally {
      setBusy(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Normalized record" }]} />
      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Normalized record</h1>
          <p className="mt-1 text-sm text-neutral-600">
            Permission-scoped evidence. API <code className="rounded bg-neutral-100 px-1">{apiBase()}</code> · principal{" "}
            <code className="rounded bg-neutral-100 px-1">{principalUserId()}</code>
          </p>
          <p className="mt-2 text-sm text-neutral-600">
            ID: <code className="rounded bg-neutral-100 px-1">{id}</code>
          </p>
        </div>
        <div className="flex gap-3 text-sm">
          <Link href="/search" className="text-blue-700 underline">
            Search
          </Link>
          <Link href="/governance" className="text-blue-700 underline">
            Governance
          </Link>
        </div>
      </div>

      {err ? (
        <div className="mb-6 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div>
      ) : null}

      {rec ? (
        <section className="mb-6 rounded-lg border border-neutral-200 bg-white p-4 text-sm">
          <h2 className="text-sm font-medium text-neutral-900">Connector metadata</h2>
          <dl className="mt-3 grid gap-2 text-xs text-neutral-800 sm:grid-cols-2">
            <div>
              <dt className="text-neutral-500">Record type</dt>
              <dd>{rec.record_type}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Raw artifact</dt>
              <dd>
                <Link href={`/raw-artifacts/${rec.raw_artifact_id}`} className="font-mono text-blue-700 underline">
                  {rec.raw_artifact_id}
                </Link>
              </dd>
            </div>
            <div>
              <dt className="text-neutral-500">Source feed</dt>
              <dd className="font-mono">{rec.source_feed_id}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Detected author</dt>
              <dd className="break-all">{rec.detected_author_ref ?? "—"}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Source time</dt>
              <dd>{rec.source_timestamp ? new Date(rec.source_timestamp).toISOString() : "—"}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Normalized at</dt>
              <dd>{new Date(rec.created_at).toISOString()}</dd>
            </div>
          </dl>
        </section>
      ) : null}

      <button
        type="button"
        className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy}
        onClick={load}
      >
        {busy ? "Refreshing…" : "Refresh"}
      </button>

      <section className="mt-6">
        <h2 className="text-lg font-medium">Record JSON</h2>
        <pre className="mt-2 max-h-[520px] overflow-auto rounded bg-neutral-50 p-3 text-xs">
          {rec ? JSON.stringify(rec, null, 2) : "—"}
        </pre>
      </section>
    </main>
  );
}

