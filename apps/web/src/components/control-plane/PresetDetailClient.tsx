"use client";

/**
 * Native CP Preset detail + instantiation (Phase 2.1.4).
 *
 * Reads GET /api/presets/:id (entry + categories + preview), GET /api/presets/:id/related,
 * and posts to POST /api/presets/:id/instantiate to create a working copy.
 * On success the response may include `edit_path_hint` linking to the new live object.
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiJson, formatApiClientError } from "@/lib/api";

type Json = Record<string, unknown>;

type RelatedEntry = {
  relationship_type: string;
  entry: { id: string; preset_type: string; code: string; name: string; active: boolean };
};

type Detail = {
  entry: { id: string; preset_type: string; code: string; name: string; description?: string | null; active: boolean };
  categories: { axis: string; code: string; label: string }[];
  preview: unknown;
};

export function PresetDetailClient({ id }: { id: string }) {
  const [err, setErr] = useState<string | null>(null);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [related, setRelated] = useState<RelatedEntry[]>([]);
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [inst, setInst] = useState<Json | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setBusy(true);
    setErr(null);
    try {
      const [d, rel] = await Promise.all([
        apiJson<Detail>(`/api/presets/${encodeURIComponent(id)}`),
        apiJson<RelatedEntry[]>(`/api/presets/${encodeURIComponent(id)}/related`),
      ]);
      setDetail(d);
      setRelated(rel);
      setName(`${d.entry.name} (copy)`);
      setCode(`${d.entry.code}_copy`);
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  async function instantiate() {
    if (!id) return;
    setBusy(true);
    setErr(null);
    setInst(null);
    try {
      const body: Record<string, string> = {};
      if (name.trim()) body.name = name.trim();
      if (code.trim()) body.code = code.trim();
      const res = await apiJson<Json>(`/api/presets/${encodeURIComponent(id)}/instantiate`, {
        method: "POST",
        body: JSON.stringify(body),
      });
      setInst(res);
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }

  function editHint(row: Json | null) {
    const hint = row && typeof row.edit_path_hint === "string" ? row.edit_path_hint : null;
    if (!hint) return null;
    return (
      <Link href={hint} className="text-sm text-blue-700 underline hover:text-blue-900">
        Open instantiated object →
      </Link>
    );
  }

  if (err) {
    return <div className="mt-6 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{err}</div>;
  }
  if (!detail) {
    return <p className="mt-6 text-sm text-gray-500">{busy ? "Loading…" : "No data."}</p>;
  }

  return (
    <div className="mt-6 grid gap-6 lg:grid-cols-2">
      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h2 className="text-lg font-medium text-gray-900">Entry</h2>
        <dl className="mt-3 space-y-2 text-sm">
          <Term k="Type" v={<span className="font-mono text-xs">{detail.entry.preset_type}</span>} />
          <Term k="Code" v={<span className="font-mono text-xs">{detail.entry.code}</span>} />
          <Term k="Name" v={detail.entry.name} />
          {detail.entry.description ? <Term k="Description" v={detail.entry.description} /> : null}
        </dl>
        <h3 className="mt-4 text-sm font-medium text-gray-700">Categories</h3>
        <ul className="mt-1 text-xs text-gray-600">
          {detail.categories?.length
            ? detail.categories.map((c) => (
                <li key={`${c.axis}-${c.code}`}>
                  <span className="font-mono">{c.axis}:{c.code}</span> ({c.label})
                </li>
              ))
            : <li>—</li>}
        </ul>
      </section>

      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h2 className="text-lg font-medium text-gray-900">Related</h2>
        <ul className="mt-3 max-h-48 space-y-2 overflow-auto text-sm">
          {related.length === 0 ? (
            <li className="text-gray-500">No outgoing relationships.</li>
          ) : (
            related.map((r) => (
              <li key={r.entry.id}>
                <span className="text-xs text-gray-500">{r.relationship_type}</span>{" "}
                <Link href={`/control-plane/presets/${encodeURIComponent(r.entry.id)}`} className="text-blue-700 underline hover:text-blue-900">
                  {r.entry.preset_type}:{r.entry.code}
                </Link>
              </li>
            ))
          )}
        </ul>
      </section>

      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm lg:col-span-2">
        <h2 className="text-lg font-medium text-gray-900">Preview</h2>
        <pre className="mt-2 max-h-64 overflow-auto rounded bg-gray-50 p-3 text-xs">
          {JSON.stringify(detail.preview, null, 2)}
        </pre>
      </section>

      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm lg:col-span-2">
        <h2 className="text-lg font-medium text-gray-900">Instantiate</h2>
        <p className="mt-1 text-xs text-gray-500">
          Creates a new role, scenario, or job from this catalog entry. The created object is governed and editable via the matching builder.
        </p>
        <form
          className="mt-3 flex flex-wrap gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            void instantiate();
          }}
        >
          <label className="flex flex-col text-xs text-gray-600">
            Name
            <input value={name} onChange={(e) => setName(e.target.value)} className="mt-1 rounded-md border border-gray-300 px-2 py-1 text-sm" />
          </label>
          <label className="flex flex-col text-xs text-gray-600">
            Code
            <input value={code} onChange={(e) => setCode(e.target.value)} className="mt-1 rounded-md border border-gray-300 px-2 py-1 text-sm" />
          </label>
          <button type="submit" disabled={busy} className="self-end rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400">
            {busy ? "Working…" : "Instantiate"}
          </button>
        </form>
        {inst ? (
          <div className="mt-4 rounded-md border border-green-200 bg-green-50 p-3 text-sm">
            <pre className="overflow-auto text-xs">{JSON.stringify(inst, null, 2)}</pre>
            <div className="mt-2">{editHint(inst)}</div>
          </div>
        ) : null}
      </section>
    </div>
  );
}

function Term({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <div>
      <dt className="text-gray-500">{k}</dt>
      <dd>{v}</dd>
    </div>
  );
}
