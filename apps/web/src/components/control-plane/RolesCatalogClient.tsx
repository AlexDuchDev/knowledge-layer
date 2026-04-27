"use client";

/**
 * Native CP Roles catalog (Phase 2.1.1 — replaces middleware-rewrite to legacy /admin/roles).
 *
 * Read-only catalog over the existing role_builder API: lists roles, presets,
 * and per-role detail (definition + access preview + assignments). Full
 * create/edit forms remain a follow-up; this is the consolidation step that
 * removes the rewrite and ships the catalog under canonical /control-plane URLs.
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiJson, formatApiClientError } from "@/lib/api";

type RoleSummary = {
  id: string;
  code: string;
  name: string;
  category: string;
  active: boolean;
  scope_model: string;
  is_preset: boolean;
  preset_key?: string | null;
};

type Json = Record<string, unknown>;

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

export function RolesCatalogClient() {
  const [err, setErr] = useState<string | null>(null);
  const [roles, setRoles] = useState<RoleSummary[]>([]);
  const [presets, setPresets] = useState<RoleSummary[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detailJson, setDetailJson] = useState<string>("");
  const [previewJson, setPreviewJson] = useState<string>("");
  const [assignmentsJson, setAssignmentsJson] = useState<string>("");
  const [busy, setBusy] = useState(false);

  const loadLists = useCallback(async () => {
    setErr(null);
    try {
      const [r, p] = await Promise.all([
        apiJson<RoleSummary[]>("/roles"),
        apiJson<RoleSummary[]>("/roles/presets"),
      ]);
      setRoles(r);
      setPresets(p);
    } catch (e) {
      setErr(formatApiClientError(e));
    }
  }, []);

  useEffect(() => {
    void loadLists();
  }, [loadLists]);

  const loadDetail = useCallback(async (id: string) => {
    setBusy(true);
    setErr(null);
    try {
      const d = await apiJson<Json>(`/roles/${id}`);
      setDetailJson(JSON.stringify(d, null, 2));
      const pv = await apiJson<Json>(`/roles/${id}/preview`);
      setPreviewJson(JSON.stringify(pv, null, 2));
      const asg = await apiJson<Json[]>(`/roles/${id}/assignments`);
      setAssignmentsJson(JSON.stringify(asg, null, 2));
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    if (selectedId) void loadDetail(selectedId);
  }, [selectedId, loadDetail]);

  return (
    <div className="mt-6 space-y-8">
      {err ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{err}</div>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <div className="flex items-baseline justify-between">
            <h2 className="text-lg font-medium text-gray-900">Roles</h2>
            <Link href="/control-plane/roles/new" className="text-sm text-blue-700 underline hover:text-blue-900">
              + New role
            </Link>
          </div>
          <p className="mt-1 text-xs text-gray-500">{roles.length} role(s). Select to load detail / access preview / assignments.</p>
          <ul className="mt-3 max-h-72 divide-y divide-gray-100 overflow-auto text-sm">
            {roles.length === 0 ? (
              <li className="px-2 py-4 text-gray-500">No roles defined yet. Create one or instantiate from a preset.</li>
            ) : (
              roles.map((r) => (
                <li key={r.id}>
                  <button
                    type="button"
                    className={`w-full rounded px-2 py-2 text-left transition hover:bg-gray-50 ${selectedId === r.id ? "bg-blue-50 font-medium" : ""}`}
                    onClick={() => setSelectedId(r.id)}
                  >
                    <span className="block">{r.name}</span>
                    <span className="text-xs text-gray-500">
                      <code className="font-mono">{r.code}</code> · {r.category} · {r.active ? "active" : "inactive"}
                      {r.is_preset ? " · preset" : ""}
                    </span>
                  </button>
                </li>
              ))
            )}
          </ul>
        </section>

        <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium text-gray-900">Presets</h2>
          <p className="mt-1 text-xs text-gray-500">
            Preset templates ready to clone via <code className="font-mono">POST /roles/from-preset</code>.
          </p>
          <ul className="mt-3 max-h-72 divide-y divide-gray-100 overflow-auto text-sm">
            {presets.length === 0 ? (
              <li className="px-2 py-4 text-gray-500">No presets registered.</li>
            ) : (
              presets.map((r) => (
                <li key={r.id} className="px-2 py-2">
                  <div className="font-medium">{r.name}</div>
                  <div className="text-xs text-gray-500">
                    <code className="font-mono">{asStr(r.preset_key)}</code> · {r.code}
                  </div>
                </li>
              ))
            )}
          </ul>
        </section>
      </div>

      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-medium text-gray-900">Detail · Access preview · Assignments</h2>
          {busy ? <span className="text-xs text-gray-500">Loading…</span> : null}
        </div>
        {!selectedId ? (
          <p className="mt-2 text-sm text-gray-500">Select a role above to inspect its definition, effective-access preview, and current assignments.</p>
        ) : (
          <div className="mt-4 grid gap-4 lg:grid-cols-3">
            <Pane title={`GET /roles/${selectedId.slice(0, 8)}…`} body={detailJson} />
            <Pane title={`GET /roles/${selectedId.slice(0, 8)}…/preview`} body={previewJson} />
            <Pane title={`GET /roles/${selectedId.slice(0, 8)}…/assignments`} body={assignmentsJson} />
          </div>
        )}
      </section>
    </div>
  );
}

function Pane({ title, body }: { title: string; body: string }) {
  return (
    <div>
      <h3 className="text-xs font-medium uppercase tracking-wide text-gray-700">{title}</h3>
      <pre className="mt-2 max-h-72 overflow-auto rounded bg-gray-900 p-3 text-xs text-gray-100">{body || "—"}</pre>
    </div>
  );
}
