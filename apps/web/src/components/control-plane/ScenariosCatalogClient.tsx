"use client";

/**
 * Native CP Scenarios catalog (Phase 2.1.2 — replaces middleware-rewrite to legacy /admin/scenarios).
 *
 * Lists scenarios + presets, supports the "create from preset" flow already
 * exposed by the scenario_builder API (POST /scenarios/from-preset). Scenario
 * detail (bindings: roles / sources / jobs) lives at /control-plane/scenarios/[id].
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiBase, apiHeaders, apiJson, formatApiClientError } from "@/lib/api";

type ScenarioSummary = {
  id: string;
  code: string;
  name: string;
  scenario_type: string;
  active: boolean;
  is_preset: boolean;
  preset_key?: string | null;
  visible_role_codes: string[];
};

type PresetRow = {
  preset_key: string;
  name: string;
  scenario_type: string;
};

export function ScenariosCatalogClient() {
  const [err, setErr] = useState<string | null>(null);
  const [scenarios, setScenarios] = useState<ScenarioSummary[]>([]);
  const [presets, setPresets] = useState<PresetRow[]>([]);
  const [clonePresetKey, setClonePresetKey] = useState("");
  const [cloneCode, setCloneCode] = useState("");
  const [cloneName, setCloneName] = useState("");
  const [busy, setBusy] = useState(false);

  const loadLists = useCallback(async () => {
    setErr(null);
    try {
      const [s, p] = await Promise.all([
        apiJson<ScenarioSummary[]>("/scenarios"),
        apiJson<PresetRow[]>("/scenarios/presets"),
      ]);
      setScenarios(s);
      setPresets(p);
    } catch (e) {
      setErr(formatApiClientError(e));
    }
  }, []);

  useEffect(() => {
    void loadLists();
  }, [loadLists]);

  async function createFromPreset() {
    setBusy(true);
    setErr(null);
    try {
      const res = await fetch(`${apiBase()}/scenarios/from-preset`, {
        method: "POST",
        credentials: "include",
        headers: apiHeaders(),
        body: JSON.stringify({
          preset_key: clonePresetKey.trim(),
          code: cloneCode.trim(),
          name: cloneName.trim(),
        }),
      });
      if (!res.ok) {
        throw new Error(`${res.status} ${(await res.text()).slice(0, 400)}`);
      }
      setCloneCode("");
      setCloneName("");
      await loadLists();
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-6 space-y-8">
      {err ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{err}</div>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <div className="flex items-baseline justify-between">
            <h2 className="text-lg font-medium text-gray-900">Scenarios</h2>
            <Link href="/control-plane/scenarios/new" className="text-sm text-blue-700 underline hover:text-blue-900">
              + New scenario
            </Link>
          </div>
          <p className="mt-1 text-xs text-gray-500">{scenarios.length} scenario(s). Tap a name to open bindings (roles / sources / jobs).</p>
          <ul className="mt-3 max-h-72 divide-y divide-gray-100 overflow-auto text-sm">
            {scenarios.length === 0 ? (
              <li className="px-2 py-4 text-gray-500">No scenarios yet. Clone a preset to get started.</li>
            ) : (
              scenarios.map((s) => (
                <li key={s.id} className="px-2 py-2">
                  <Link href={`/control-plane/scenarios/${encodeURIComponent(s.id)}`} className="font-medium text-blue-700 hover:underline">
                    {s.name}
                  </Link>
                  <div className="text-xs text-gray-500">
                    <code className="font-mono">{s.code}</code> · {s.scenario_type} · {s.active ? "active" : "inactive"}
                    {s.is_preset ? " · preset" : ""}
                  </div>
                  <div className="text-xs text-gray-600">
                    Visible roles: {s.visible_role_codes?.length ? s.visible_role_codes.join(", ") : "—"}
                  </div>
                </li>
              ))
            )}
          </ul>
        </section>

        <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium text-gray-900">Presets</h2>
          <p className="mt-1 text-xs text-gray-500">
            Cloning a preset creates an editable scenario via <code className="font-mono">POST /scenarios/from-preset</code>.
          </p>
          <ul className="mt-3 max-h-40 divide-y divide-gray-100 overflow-auto text-sm">
            {presets.length === 0 ? (
              <li className="px-2 py-4 text-gray-500">No presets registered.</li>
            ) : (
              presets.map((p) => (
                <li key={p.preset_key} className="flex items-baseline gap-2 px-2 py-1.5">
                  <button type="button" onClick={() => setClonePresetKey(p.preset_key)} className="font-medium text-blue-700 underline hover:text-blue-900">
                    {p.name}
                  </button>
                  <span className="text-xs text-gray-500">
                    <code className="font-mono">{p.preset_key}</code> · {p.scenario_type}
                  </span>
                </li>
              ))
            )}
          </ul>
          <form
            className="mt-4 space-y-2 border-t border-gray-100 pt-4"
            onSubmit={(e) => {
              e.preventDefault();
              void createFromPreset();
            }}
          >
            <Field label="preset_key">
              <input value={clonePresetKey} onChange={(e) => setClonePresetKey(e.target.value)} placeholder="weekly_team_digest" className="w-full rounded-md border border-gray-300 px-2 py-1 text-sm" />
            </Field>
            <Field label="code (unique)">
              <input value={cloneCode} onChange={(e) => setCloneCode(e.target.value)} placeholder="my_team_digest" className="w-full rounded-md border border-gray-300 px-2 py-1 text-sm" />
            </Field>
            <Field label="name">
              <input value={cloneName} onChange={(e) => setCloneName(e.target.value)} placeholder="My team digest" className="w-full rounded-md border border-gray-300 px-2 py-1 text-sm" />
            </Field>
            <button type="submit" disabled={busy || !clonePresetKey.trim() || !cloneCode.trim() || !cloneName.trim()} className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400">
              {busy ? "Creating…" : "Create from preset"}
            </button>
          </form>
        </section>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block text-xs font-medium text-gray-700">
      <span>{label}</span>
      <div className="mt-1">{children}</div>
    </label>
  );
}
