"use client";

/**
 * Effective access inspector (Phase 2.1.6).
 *
 * Calls GET /users/:id/effective-access?... and renders the 9-step pipeline
 * trace returned by the backend AccessEvaluator. Operators use this to debug
 * "why can't user X see entity Y" questions without grep-ing audit_events.
 *
 * Requires identity-admin caller OR the page being viewed by the user
 * themselves (backend enforces; UI does not pre-filter).
 */

import { useState } from "react";
import { apiJson, formatApiClientError } from "@/lib/api";

type EffectiveAccessResponse = {
  allow: boolean;
  reason_code: string;
  reasons: string[];
  trace: string[];
  sensitivity_ok: boolean;
  effective_sensitivity: number | null;
  resolved_domain_id: string | null;
  sensitivity_result: string;
  matched_rule_code: string;
  matched_policies?: string[];
  matched_overrides?: string[];
  input: {
    action: string;
    resource_type: string;
    resource_id: string | null;
    domain_id: string | null;
    sensitivity_level: number | null;
    entity_type: string | null;
  };
};

const ACTIONS = ["view", "search", "view_raw", "edit", "create", "archive", "export", "run_job", "manage_jobs", "manage_sources", "manage_source_feed", "approve", "review", "publish"];
const RESOURCE_TYPES = ["entity", "domain", "source_feed", "knowledge_job", "review_task"];

export function EffectiveAccessClient({ userId }: { userId: string }) {
  const [action, setAction] = useState("view");
  const [resourceType, setResourceType] = useState("entity");
  const [resourceId, setResourceId] = useState("");
  const [domainId, setDomainId] = useState("");
  const [sensitivityLevel, setSensitivityLevel] = useState("");
  const [entityType, setEntityType] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<EffectiveAccessResponse | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      const params = new URLSearchParams({ action, resource_type: resourceType });
      if (resourceId.trim()) params.set("resource_id", resourceId.trim());
      if (domainId.trim()) params.set("domain_id", domainId.trim());
      if (sensitivityLevel.trim()) params.set("sensitivity_level", sensitivityLevel.trim());
      if (entityType.trim()) params.set("entity_type", entityType.trim());
      const data = await apiJson<EffectiveAccessResponse>(`/users/${encodeURIComponent(userId)}/effective-access?${params.toString()}`);
      setResult(data);
    } catch (err) {
      setError(formatApiClientError(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mt-6 space-y-6">
      <form onSubmit={onSubmit} className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h2 className="mb-3 text-sm font-semibold text-gray-900">Evaluate access</h2>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Action" required>
            <select value={action} onChange={(e) => setAction(e.target.value)} className="w-full rounded-md border border-gray-300 px-2 py-1 text-sm">
              {ACTIONS.map((a) => (
                <option key={a} value={a}>{a}</option>
              ))}
            </select>
          </Field>
          <Field label="Resource type" required>
            <select value={resourceType} onChange={(e) => setResourceType(e.target.value)} className="w-full rounded-md border border-gray-300 px-2 py-1 text-sm">
              {RESOURCE_TYPES.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </Field>
          <Field label="Resource ID (uuid, optional)">
            <input value={resourceId} onChange={(e) => setResourceId(e.target.value)} placeholder="00000000-…" className="w-full rounded-md border border-gray-300 px-2 py-1 font-mono text-xs" />
          </Field>
          <Field label="Domain ID (uuid, optional)">
            <input value={domainId} onChange={(e) => setDomainId(e.target.value)} placeholder="00000000-…" className="w-full rounded-md border border-gray-300 px-2 py-1 font-mono text-xs" />
          </Field>
          <Field label="Sensitivity level (0–4, optional)">
            <input value={sensitivityLevel} onChange={(e) => setSensitivityLevel(e.target.value)} placeholder="0" className="w-full rounded-md border border-gray-300 px-2 py-1 text-sm" />
          </Field>
          <Field label="Entity type (optional)">
            <input value={entityType} onChange={(e) => setEntityType(e.target.value)} placeholder="decision" className="w-full rounded-md border border-gray-300 px-2 py-1 text-sm" />
          </Field>
        </div>
        <div className="mt-4 flex items-center gap-3">
          <button type="submit" disabled={loading} className="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400">
            {loading ? "Evaluating…" : "Evaluate"}
          </button>
          <p className="text-xs text-gray-500">Calls <code className="font-mono">GET /users/{userId.slice(0, 8)}…/effective-access</code></p>
        </div>
      </form>

      {error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</div>
      ) : null}

      {result ? <ResultPanel res={result} /> : null}
    </div>
  );
}

function Field({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) {
  return (
    <label className="block text-xs font-medium text-gray-700">
      <span>{label}{required ? <span className="text-red-600"> *</span> : null}</span>
      <div className="mt-1">{children}</div>
    </label>
  );
}

function ResultPanel({ res }: { res: EffectiveAccessResponse }) {
  const verdictBg = res.allow && res.sensitivity_ok ? "bg-green-50 border-green-200 text-green-900" : "bg-red-50 border-red-200 text-red-900";
  return (
    <div className="space-y-4">
      <div className={`rounded-lg border p-4 ${verdictBg}`}>
        <div className="text-sm font-semibold">
          {res.allow && res.sensitivity_ok ? "ALLOW" : "DENY"} — <span className="font-mono">{res.matched_rule_code || res.reason_code || "—"}</span>
        </div>
        {res.reasons.length > 0 ? (
          <ul className="mt-2 list-disc pl-5 text-sm">
            {res.reasons.map((r, i) => (
              <li key={i}>{r}</li>
            ))}
          </ul>
        ) : null}
        <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
          <Term k="Sensitivity OK" v={String(res.sensitivity_ok)} />
          <Term k="Sensitivity result" v={res.sensitivity_result || "—"} />
          <Term k="Effective sensitivity" v={res.effective_sensitivity != null ? String(res.effective_sensitivity) : "—"} />
          <Term k="Resolved domain" v={res.resolved_domain_id ?? "—"} />
        </dl>
      </div>

      <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h3 className="mb-2 text-sm font-semibold text-gray-900">9-step access pipeline trace</h3>
        {res.trace.length === 0 ? (
          <p className="text-sm text-gray-600">No trace returned.</p>
        ) : (
          <ol className="space-y-1 text-sm font-mono">
            {res.trace.map((step, i) => {
              const isDeny = step.toLowerCase().includes("deny");
              return (
                <li key={i} className={isDeny ? "text-red-700" : "text-gray-800"}>
                  {i + 1}. {step}
                </li>
              );
            })}
          </ol>
        )}
        <p className="mt-3 text-xs text-gray-500">Steps mirror <code className="font-mono">identity_access.AccessEvaluator.Evaluate</code> (1: principal → 9: sensitivity cap).</p>
      </div>

      {(res.matched_policies?.length || res.matched_overrides?.length) ? (
        <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
          <h3 className="mb-2 text-sm font-semibold text-gray-900">Matched rules</h3>
          {res.matched_policies?.length ? (
            <p className="text-xs">Policies: <span className="font-mono">{res.matched_policies.join(", ")}</span></p>
          ) : null}
          {res.matched_overrides?.length ? (
            <p className="text-xs">Overrides: <span className="font-mono">{res.matched_overrides.join(", ")}</span></p>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function Term({ k, v }: { k: string; v: string }) {
  return (
    <>
      <dt className="text-gray-500">{k}</dt>
      <dd className="font-mono text-gray-800">{v}</dd>
    </>
  );
}
