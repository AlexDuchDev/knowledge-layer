"use client";

/**
 * Native CP Scenario detail (Phase 2.1.2 — replaces middleware-rewrite to legacy /admin/scenarios/[id]).
 *
 * Read-only definition + preview JSON over scenario_builder API. The builder
 * sections (basic info, role/source/job bindings, output policy, etc.) remain
 * a documentation outline below for now; binding editors live under
 * /control-plane/scenarios/[id]/bindings/*.
 */

import { useCallback, useEffect, useState } from "react";
import { apiJson, formatApiClientError } from "@/lib/api";

type Json = Record<string, unknown>;

const SECTIONS: { title: string; body: string }[] = [
  {
    title: "Basic info",
    body: "code, name, description, scenario_type, active. Edited via PATCH /scenarios/:id; scenario_type classifies for UI/routing only.",
  },
  {
    title: "Intended users / roles",
    body: "POST /scenarios/:id/role-bindings — bind roles whose can_see flag should grant access to outputs.",
  },
  {
    title: "Input scope",
    body: "input_scope_json on definition. Restricts which source feeds and entity types the scenario draws from.",
  },
  {
    title: "Trigger & timing",
    body: "trigger_type and trigger_config_json describe how the scenario runs (schedule, manual, event). Must match enabled triggers in the API.",
  },
  {
    title: "Processing mode",
    body: "processing_mode controls synchronous vs queued execution; queued requires Redis.",
  },
  {
    title: "Output configuration",
    body: "output_mode + ui_surface choose how the scenario surfaces its results (digest doc, entity, content hub, …).",
  },
  {
    title: "Governance rules",
    body: "output_policy constrains what may be emitted (draft vs publish, redaction). Align with domain sensitivity and review queues.",
  },
  {
    title: "Linked jobs",
    body: "POST /scenarios/:id/job-bindings — connect knowledge jobs that produce/consume this scenario.",
  },
];

export function ScenarioDetailClient({ id }: { id: string }) {
  const [err, setErr] = useState<string | null>(null);
  const [detailJson, setDetailJson] = useState<string>("");
  const [previewJson, setPreviewJson] = useState<string>("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    setBusy(true);
    setErr(null);
    try {
      const d = await apiJson<Json>(`/scenarios/${encodeURIComponent(id)}`);
      setDetailJson(JSON.stringify(d, null, 2));
      const pv = await apiJson<Json>(`/scenarios/${encodeURIComponent(id)}/preview`);
      setPreviewJson(JSON.stringify(pv, null, 2));
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="mt-6 space-y-6">
      {err ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{err}</div>
      ) : null}
      {busy ? <p className="text-sm text-gray-500">Loading…</p> : null}

      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h2 className="text-lg font-medium text-gray-900">Builder sections</h2>
        <p className="mt-1 text-xs text-gray-500">
          Reference outline of what this scenario captures. Field-level edit forms are progressively replacing the raw JSON view.
        </p>
        <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm text-gray-800">
          {SECTIONS.map((s) => (
            <li key={s.title}>
              <span className="font-medium">{s.title}</span> — {s.body}
            </li>
          ))}
        </ol>
      </section>

      <div className="grid gap-6 lg:grid-cols-2">
        <Pane title={`GET /scenarios/${id.slice(0, 8)}…`} body={detailJson} />
        <Pane title={`GET /scenarios/${id.slice(0, 8)}…/preview`} body={previewJson} />
      </div>
    </div>
  );
}

function Pane({ title, body }: { title: string; body: string }) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <h3 className="text-xs font-medium uppercase tracking-wide text-gray-700">{title}</h3>
      <pre className="mt-2 max-h-[28rem] overflow-auto rounded bg-gray-900 p-3 text-xs text-gray-100">{body || "—"}</pre>
    </section>
  );
}
