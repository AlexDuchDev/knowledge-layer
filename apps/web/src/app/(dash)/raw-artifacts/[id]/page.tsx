"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { apiBase, apiJson, principalUserId } from "@/lib/api";

type RawArtifact = {
  id: string;
  source_feed_id: string;
  domain_id: string;
  feed_sensitivity_level: number;
  ingestion_run_id: string;
  artifact_type: string;
  external_artifact_id?: string | null;
  storage_uri: string;
  content_hash: string;
  source_created_at?: string | null;
  source_author_ref?: string | null;
  metadata_json?: unknown;
  created_at: string;
};

export default function RawArtifactDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const [err, setErr] = useState<string | null>(null);
  const [raw, setRaw] = useState<RawArtifact | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setErr(null);
    setBusy(true);
    try {
      if (!id) throw new Error("missing id");
      const out = await apiJson<RawArtifact>(`/raw-artifacts/${id}`);
      setRaw(out);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setRaw(null);
    } finally {
      setBusy(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  const metadataPretty = useMemo(() => {
    if (raw?.metadata_json == null) return "";
    try {
      return JSON.stringify(raw.metadata_json, null, 2);
    } catch {
      return String(raw.metadata_json);
    }
  }, [raw?.metadata_json]);

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Raw artifact" }]} />
      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Raw artifact</h1>
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

      {raw ? (
        <section className="mb-6 rounded-lg border border-neutral-200 bg-white p-4 text-sm">
          <h2 className="text-sm font-medium text-neutral-900">Connector metadata</h2>
          <dl className="mt-3 grid gap-2 text-xs text-neutral-800 sm:grid-cols-2">
            <div>
              <dt className="text-neutral-500">Source feed</dt>
              <dd className="font-mono">{raw.source_feed_id}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Artifact type</dt>
              <dd>{raw.artifact_type}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">External ref</dt>
              <dd className="break-all font-mono">{raw.external_artifact_id ?? "—"}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Author ref</dt>
              <dd className="break-all">{raw.source_author_ref ?? "—"}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Source time</dt>
              <dd>{raw.source_created_at ? new Date(raw.source_created_at).toISOString() : "—"}</dd>
            </div>
            <div>
              <dt className="text-neutral-500">Ingested at</dt>
              <dd>{new Date(raw.created_at).toISOString()}</dd>
            </div>
            <div className="sm:col-span-2">
              <dt className="text-neutral-500">Storage URI</dt>
              <dd className="break-all font-mono text-[11px]">{raw.storage_uri}</dd>
            </div>
          </dl>
          {metadataPretty ? (
            <div className="mt-4">
              <div className="text-xs font-medium text-neutral-700">metadata_json</div>
              <pre className="mt-1 max-h-48 overflow-auto rounded bg-neutral-50 p-2 text-[11px]">{metadataPretty}</pre>
            </div>
          ) : null}
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
        <h2 className="text-lg font-medium">Artifact JSON</h2>
        <pre className="mt-2 max-h-[520px] overflow-auto rounded bg-neutral-50 p-3 text-xs">
          {raw ? JSON.stringify(raw, null, 2) : "—"}
        </pre>
      </section>
    </main>
  );
}

