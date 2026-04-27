"use client";

import { useCallback, useState } from "react";
import Link from "next/link";
import { apiJson, formatApiClientError } from "@/lib/api";

type Json = Record<string, unknown>;

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

export default function ConnectorsPage() {
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [list, setList] = useState<Json[] | null>(null);
  const [ctype, setCtype] = useState("");
  const [dname, setDname] = useState("");

  const run = useCallback(async (fn: () => Promise<void>) => {
    setErr(null);
    setBusy(true);
    try {
      await fn();
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }, []);

  return (
    <main className="mx-auto max-w-4xl px-6 py-10">
      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Connectors</h1>
          <p className="mt-1 max-w-2xl text-sm text-neutral-600">
            Register <strong>connector types</strong> your instance recognizes. That alone does not deliver ingestion: you still create{" "}
            <Link href="/control-plane/sources" className="text-blue-700 underline">
              source feeds
            </Link>{" "}
            and run syncs. Normalization coverage varies by family and artifact type — see{" "}
            <span className="font-mono text-neutral-800">docs/CONNECTOR_CAPABILITY_MATRIX.md</span> and{" "}
            <span className="font-mono text-neutral-800">docs/LIMITATIONS.md</span> in the repository.
          </p>
        </div>
        <div className="flex gap-3 text-sm">
          <Link href="/" className="text-blue-700 underline">
            Home
          </Link>
          <Link href="/control-plane/sources" className="text-blue-700 underline">
            Source feeds
          </Link>
        </div>
      </div>
      {err ? <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy}
          onClick={() =>
            run(async () => {
              setList(await apiJson<Json[]>("/connectors"));
            })
          }
        >
          {busy ? "…" : "List connectors"}
        </button>
      </div>

      {list ? (
        <ul className="mt-4 space-y-2 text-sm">
          {list.map((c) => (
            <li key={asStr(c.id)} className="rounded border border-neutral-200 px-3 py-2 font-mono text-xs">
              {asStr(c.display_name)} · {asStr(c.type)} · {asStr(c.status)}
            </li>
          ))}
        </ul>
      ) : null}

      <section className="mt-10 rounded-lg border border-neutral-200 p-4">
        <h2 className="text-sm font-medium">Register a connector type</h2>
        <p className="mt-1 text-xs text-neutral-600">Unique type string and display name. Requires identity-admin permissions.</p>
        <div className="mt-3 flex max-w-lg flex-col gap-2 sm:flex-row">
          <input className="flex-1 rounded border px-2 py-1 text-sm" placeholder="type" value={ctype} onChange={(e) => setCtype(e.target.value)} />
          <input className="flex-1 rounded border px-2 py-1 text-sm" placeholder="display_name" value={dname} onChange={(e) => setDname(e.target.value)} />
        </div>
        <button
          type="button"
          className="mt-2 rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy || !ctype.trim() || !dname.trim()}
          onClick={() =>
            run(async () => {
              await apiJson("/connectors", {
                method: "POST",
                body: JSON.stringify({ type: ctype.trim(), display_name: dname.trim() }),
              });
              setCtype("");
              setDname("");
              setList(await apiJson<Json[]>("/connectors"));
            })
          }
        >
          Register
        </button>
        <p className="mt-2 text-[11px] text-neutral-500">Advanced: same action as POST /connectors on the API.</p>
      </section>
    </main>
  );
}
