"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { Suspense, useCallback, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { FromCpBanner } from "@/components/shell/FromCpBanner";
import { apiJson } from "@/lib/api";

function SyncInner() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const [msg, setMsg] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const run = useCallback(async () => {
    if (!id) return;
    setBusy(true);
    setErr(null);
    setMsg(null);
    try {
      const out = await apiJson<Record<string, unknown>>(`/source-feeds/${encodeURIComponent(id)}/sync`, { method: "POST" });
      setMsg(JSON.stringify(out, null, 2).slice(0, 2000));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [id]);

  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <Suspense fallback={null}>
        <FromCpBanner />
      </Suspense>
      <AppBreadcrumb
        items={[
          { label: "Home", href: "/" },
          { label: "Source feeds", href: "/source-feeds" },
          { label: "Sync" },
        ]}
      />
      <h1 className="mt-4 text-2xl font-semibold tracking-tight">Manual sync</h1>
      <p className="mt-2 text-sm text-neutral-600">
        Triggers <code className="rounded bg-neutral-100 px-1">POST /source-feeds/:id/sync</code> for feed{" "}
        <code className="rounded bg-neutral-100 px-1">{id || "—"}</code>.
      </p>
      <div className="mt-6 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
          disabled={!id || busy}
          onClick={() => void run()}
        >
          {busy ? "Running…" : "Run sync now"}
        </button>
        <Link href={`/control-plane/sources/feeds/${encodeURIComponent(id)}`} className="rounded-md border border-neutral-300 px-4 py-2 text-sm">
          Feed JSON detail
        </Link>
        <Link href="/control-plane/sources" className="rounded-md border border-neutral-300 px-4 py-2 text-sm">
          Sources hub
        </Link>
      </div>
      {err ? <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      {msg ? (
        <pre className="mt-4 max-h-96 overflow-auto rounded border bg-neutral-50 p-3 text-xs text-neutral-800">{msg}</pre>
      ) : null}
    </main>
  );
}

export default function SourceFeedSyncPage() {
  return (
    <Suspense fallback={<p className="p-6 text-sm text-neutral-600">Loading…</p>}>
      <SyncInner />
    </Suspense>
  );
}
