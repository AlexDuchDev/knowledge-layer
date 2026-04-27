"use client";

/**
 * Native CP Presets catalog (Phase 2.1.4 — replaces middleware-rewrite to legacy /admin/presets).
 *
 * Lists role/scenario/job presets with axis+code filters. Detail + instantiation
 * lives at /control-plane/presets/[id]. Backed by GET /api/presets.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { apiJson, formatApiClientError } from "@/lib/api";

type CategoryRef = { axis: string; code: string; label: string };

type ListRow = {
  id: string;
  preset_type: string;
  code: string;
  name: string;
  description?: string | null;
  active: boolean;
  categories: CategoryRef[];
};

const TYPES = ["", "role", "scenario", "job"];

export function PresetsCatalogClient() {
  const [err, setErr] = useState<string | null>(null);
  const [typeFilter, setTypeFilter] = useState<string>("");
  const [axis, setAxis] = useState<string>("");
  const [catCode, setCatCode] = useState<string>("");
  const [rows, setRows] = useState<ListRow[]>([]);
  const [busy, setBusy] = useState(false);

  const query = useMemo(() => {
    const p = new URLSearchParams();
    if (typeFilter) p.set("type", typeFilter);
    if (axis) p.set("category_axis", axis);
    if (catCode) p.set("category_code", catCode);
    const s = p.toString();
    return s ? `?${s}` : "";
  }, [typeFilter, axis, catCode]);

  const load = useCallback(async () => {
    setBusy(true);
    setErr(null);
    try {
      const data = await apiJson<ListRow[]>(`/api/presets${query}`);
      setRows(data);
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }, [query]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="mt-6 space-y-6">
      {err ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{err}</div>
      ) : null}

      <section className="flex flex-wrap items-end gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <Filter label="Type">
          <select value={typeFilter} onChange={(e) => setTypeFilter(e.target.value)} className="rounded-md border border-gray-300 px-2 py-1 text-sm">
            {TYPES.map((t) => (
              <option key={t} value={t}>{t || "all"}</option>
            ))}
          </select>
        </Filter>
        <Filter label="Category axis">
          <input value={axis} onChange={(e) => setAxis(e.target.value)} placeholder="e.g. function" className="rounded-md border border-gray-300 px-2 py-1 text-sm" />
        </Filter>
        <Filter label="Category code">
          <input value={catCode} onChange={(e) => setCatCode(e.target.value)} placeholder="e.g. leadership" className="rounded-md border border-gray-300 px-2 py-1 text-sm" />
        </Filter>
        <button onClick={load} disabled={busy} className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400">
          {busy ? "Loading…" : "Refresh"}
        </button>
      </section>

      <section className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-gray-200 bg-gray-50 text-xs uppercase text-gray-500">
            <tr>
              <Th>Type</Th>
              <Th>Code</Th>
              <Th>Name</Th>
              <Th>Categories</Th>
              <Th />
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {rows.length === 0 && !busy ? (
              <tr>
                <td colSpan={5} className="px-4 py-6 text-center text-sm text-gray-500">No presets match the current filters.</td>
              </tr>
            ) : (
              rows.map((r) => (
                <tr key={r.id}>
                  <Td><span className="font-mono text-xs">{r.preset_type}</span></Td>
                  <Td><span className="font-mono text-xs">{r.code}</span></Td>
                  <Td>{r.name}</Td>
                  <Td>
                    <span className="text-xs text-gray-600">
                      {r.categories?.length ? r.categories.map((c) => `${c.axis}:${c.code}`).join(", ") : "—"}
                    </span>
                  </Td>
                  <Td>
                    <Link href={`/control-plane/presets/${encodeURIComponent(r.id)}`} className="text-sm text-blue-700 underline hover:text-blue-900">
                      Open
                    </Link>
                  </Td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </section>
    </div>
  );
}

function Filter({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col text-xs font-medium text-gray-700">
      <span>{label}</span>
      <div className="mt-1">{children}</div>
    </label>
  );
}

function Th({ children }: { children?: React.ReactNode }) {
  return <th className="px-4 py-2">{children}</th>;
}
function Td({ children }: { children?: React.ReactNode }) {
  return <td className="px-4 py-2">{children}</td>;
}
