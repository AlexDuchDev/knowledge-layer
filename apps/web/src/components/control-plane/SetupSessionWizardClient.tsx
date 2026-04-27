"use client";

/**
 * Native CP setup-session wizard (Phase 2.1.5).
 *
 * Drives the onboarding session through:
 *   1. Pick a template       → POST /api/onboarding/sessions/:id/select-template
 *   2. Configure connectors  → PATCH /api/onboarding/sessions/:id (connector_selections)
 *   3. Initial admin user    → PATCH /api/onboarding/sessions/:id (assignment)
 *   4. Preview               → POST  /api/onboarding/sessions/:id/preview
 *   5. Launch                → POST  /api/onboarding/sessions/:id/launch
 *
 * Each step is committed independently so the operator can stop and resume.
 * Session state is the source of truth — the UI never holds long-lived form
 * state; after every PATCH we reload the session to surface server-side
 * validation immediately.
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiJson, formatApiClientError, principalUserId } from "@/lib/api";

type Json = Record<string, unknown>;

type Template = {
  id: string;
  code: string;
  title: string;
  description: string;
};

type ConnectorRow = {
  connector_family_code: string;
  enabled: boolean;
};

type AssignmentRow = {
  initial_admin_user_id?: string | null;
  domain_owner_user_id?: string | null;
};

type SessionView = {
  id: string;
  status: string;
  template_code?: string | null;
  org_profile_json: Json;
  steps: Record<string, Json>;
  selected_presets: Json[];
  connector_selections: ConnectorRow[];
  source_feed_drafts: Json[];
  assignment_drafts?: AssignmentRow | null;
  created_at: string;
  updated_at: string;
};

type LaunchPreview = {
  template_code?: string | null;
  validation_issues: string[];
  planned_roles: { code: string; name: string }[];
  planned_scenarios: { code: string; name: string }[];
  planned_jobs: { code: string; name: string }[];
  connectors_enabled: string[];
  assignments?: AssignmentRow | null;
};

type LaunchResult = {
  session_id: string;
  status: string;
  created: { role_ids: string[]; scenario_ids: string[]; job_ids: string[] };
  launch_log_id: string;
};

const KNOWN_FAMILIES = ["chat", "docs_wiki", "meeting", "email", "work_mgmt", "crm_support", "microsoft365"];

export function SetupSessionWizardClient({ sessionId }: { sessionId: string }) {
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [session, setSession] = useState<SessionView | null>(null);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [pickedTemplate, setPickedTemplate] = useState("");
  const [adminId, setAdminId] = useState("");
  const [preview, setPreview] = useState<LaunchPreview | null>(null);
  const [launch, setLaunch] = useState<LaunchResult | null>(null);

  const reloadSession = useCallback(async () => {
    if (!sessionId) return;
    try {
      const s = await apiJson<SessionView>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}`);
      setSession(s);
      if (!pickedTemplate && s.template_code) setPickedTemplate(s.template_code);
      if (!adminId && s.assignment_drafts?.initial_admin_user_id) {
        setAdminId(s.assignment_drafts.initial_admin_user_id);
      } else if (!adminId) {
        setAdminId(principalUserId());
      }
    } catch (e) {
      setErr(formatApiClientError(e));
    }
  }, [sessionId, pickedTemplate, adminId]);

  const loadTemplates = useCallback(async () => {
    try {
      const t = await apiJson<Template[]>("/api/onboarding/templates");
      setTemplates(t);
    } catch (e) {
      setErr(formatApiClientError(e));
    }
  }, []);

  useEffect(() => {
    void reloadSession();
    void loadTemplates();
  }, [reloadSession, loadTemplates]);

  async function selectTemplate() {
    if (!pickedTemplate.trim()) return;
    setBusy(true);
    setErr(null);
    try {
      await apiJson<SessionView>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}/select-template`, {
        method: "POST",
        body: JSON.stringify({ template_code: pickedTemplate.trim() }),
      });
      await reloadSession();
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }

  async function toggleConnector(family: string, enabled: boolean) {
    setBusy(true);
    setErr(null);
    try {
      await apiJson<SessionView>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}`, {
        method: "PATCH",
        body: JSON.stringify({
          connector_selections: [{ connector_family_code: family, enabled }],
        }),
      });
      await reloadSession();
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }

  async function saveAdmin() {
    setBusy(true);
    setErr(null);
    try {
      await apiJson<SessionView>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}`, {
        method: "PATCH",
        body: JSON.stringify({
          assignment: { initial_admin_user_id: adminId.trim() || null },
        }),
      });
      await reloadSession();
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }

  async function runPreview() {
    setBusy(true);
    setErr(null);
    setLaunch(null);
    try {
      const p = await apiJson<LaunchPreview>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}/preview`, { method: "POST" });
      setPreview(p);
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }

  async function runLaunch() {
    setBusy(true);
    setErr(null);
    try {
      const r = await apiJson<LaunchResult>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}/launch`, { method: "POST" });
      setLaunch(r);
      await reloadSession();
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }

  const enabledFamilies = new Set((session?.connector_selections ?? []).filter((c) => c.enabled).map((c) => c.connector_family_code));
  const isLaunched = session?.status === "launched";
  const blockingIssues = preview?.validation_issues ?? [];
  const canLaunch = !isLaunched && session?.template_code && (preview ? blockingIssues.length === 0 : false);

  return (
    <div className="mt-6 space-y-6">
      <div className="flex items-center gap-3 text-sm">
        <span className="rounded-full bg-gray-100 px-2.5 py-0.5 font-mono text-xs">{session?.status ?? "loading"}</span>
        <span className="font-mono text-xs text-gray-500">{sessionId}</span>
        <Link href="/control-plane/setup" className="ml-auto text-blue-700 underline hover:text-blue-900">
          ← Setup hub
        </Link>
      </div>

      {err ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{err}</div>
      ) : null}

      <Step n={1} title="Pick a template">
        <p className="text-sm text-gray-600">
          A template seeds the session with role / scenario / job presets and connector toggles. You can still customize after the choice.
        </p>
        <div className="mt-3 flex flex-wrap items-end gap-3">
          <label className="flex flex-col text-xs font-medium text-gray-700">
            <span>Template</span>
            <select
              value={pickedTemplate}
              onChange={(e) => setPickedTemplate(e.target.value)}
              className="mt-1 rounded-md border border-gray-300 px-2 py-1 text-sm"
            >
              <option value="">— pick —</option>
              {templates.map((t) => (
                <option key={t.id} value={t.code}>
                  {t.code} — {t.title}
                </option>
              ))}
            </select>
          </label>
          <button
            onClick={selectTemplate}
            disabled={busy || !pickedTemplate.trim()}
            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400"
          >
            {session?.template_code === pickedTemplate ? "Re-apply" : "Apply template"}
          </button>
          {session?.template_code ? (
            <span className="text-xs text-gray-600">
              Current: <span className="font-mono">{session.template_code}</span>
            </span>
          ) : null}
        </div>
      </Step>

      <Step n={2} title="Enable connector families">
        <p className="text-sm text-gray-600">
          Toggle the connector families this organization will use. You can wire individual source feeds later via{" "}
          <Link href="/control-plane/sources" className="text-blue-700 underline">
            Sources
          </Link>
          .
        </p>
        <div className="mt-3 flex flex-wrap gap-2">
          {KNOWN_FAMILIES.map((fam) => {
            const enabled = enabledFamilies.has(fam);
            return (
              <button
                key={fam}
                disabled={busy}
                onClick={() => toggleConnector(fam, !enabled)}
                className={`rounded-full border px-3 py-1 text-xs font-medium ${
                  enabled
                    ? "border-blue-300 bg-blue-100 text-blue-900"
                    : "border-gray-300 bg-white text-gray-700 hover:bg-gray-50"
                }`}
              >
                {enabled ? "✓ " : ""}
                {fam}
              </button>
            );
          })}
        </div>
      </Step>

      <Step n={3} title="Initial admin">
        <p className="text-sm text-gray-600">
          The user who will own the launched roles + scenarios. Defaults to the current principal — change if you are setting up on someone else&apos;s
          behalf.
        </p>
        <div className="mt-3 flex flex-wrap items-end gap-3">
          <label className="flex flex-col text-xs font-medium text-gray-700">
            <span>User UUID</span>
            <input
              value={adminId}
              onChange={(e) => setAdminId(e.target.value)}
              className="mt-1 w-80 rounded-md border border-gray-300 px-2 py-1 font-mono text-sm"
            />
          </label>
          <button onClick={saveAdmin} disabled={busy} className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400">
            Save
          </button>
        </div>
      </Step>

      <Step n={4} title="Preview the launch">
        <p className="text-sm text-gray-600">Dry-run the plan: validate the template + selections and see exactly which roles, scenarios, and jobs will be created.</p>
        <div className="mt-3 flex gap-3">
          <button onClick={runPreview} disabled={busy || !session?.template_code} className="rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-50 disabled:opacity-50">
            {busy ? "Working…" : "Run preview"}
          </button>
          {preview ? (
            <span className="text-xs text-gray-600">
              {preview.planned_roles.length} role(s), {preview.planned_scenarios.length} scenario(s), {preview.planned_jobs.length} job(s) planned
            </span>
          ) : null}
        </div>
        {preview ? (
          <div className="mt-3 space-y-3">
            {blockingIssues.length > 0 ? (
              <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm">
                <div className="font-medium text-amber-900">Blocking validation issues:</div>
                <ul className="mt-1 list-disc pl-5 text-xs text-amber-900">
                  {blockingIssues.map((iss, i) => (
                    <li key={i}>{iss}</li>
                  ))}
                </ul>
              </div>
            ) : (
              <div className="rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-900">No validation issues — ready to launch.</div>
            )}
            <PlanList title="Roles" items={preview.planned_roles} />
            <PlanList title="Scenarios" items={preview.planned_scenarios} />
            <PlanList title="Jobs" items={preview.planned_jobs} />
          </div>
        ) : null}
      </Step>

      <Step n={5} title="Launch">
        <p className="text-sm text-gray-600">
          Creates the planned roles, scenarios, and jobs in your instance. After launch the session moves to <code className="font-mono">launched</code>{" "}
          and is read-only.
        </p>
        <div className="mt-3 flex items-center gap-3">
          <button onClick={runLaunch} disabled={!canLaunch || busy} className="rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:bg-gray-400">
            {busy ? "Launching…" : "Launch"}
          </button>
          {!canLaunch && !isLaunched ? (
            <span className="text-xs text-gray-500">Run a clean preview (no validation issues) to enable launch.</span>
          ) : null}
          {isLaunched ? <span className="text-xs text-green-700">Already launched.</span> : null}
        </div>
        {launch ? (
          <div className="mt-3 rounded-md border border-green-200 bg-green-50 p-3 text-sm">
            <div className="font-medium text-green-900">Launch complete — log {launch.launch_log_id}</div>
            <p className="mt-1 text-xs text-green-900">
              Created {launch.created.role_ids.length} role(s), {launch.created.scenario_ids.length} scenario(s), {launch.created.job_ids.length} job(s).
              Open the matching CP catalog to inspect.
            </p>
          </div>
        ) : null}
      </Step>

      <details className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <summary className="cursor-pointer text-xs font-medium text-gray-600">Raw session JSON</summary>
        <pre className="mt-2 max-h-96 overflow-auto rounded bg-gray-900 p-3 text-xs text-gray-100">{session ? JSON.stringify(session, null, 2) : "—"}</pre>
      </details>
    </div>
  );
}

function Step({ n, title, children }: { n: number; title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <h2 className="text-base font-medium text-gray-900">
        <span className="mr-2 inline-flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white">{n}</span>
        {title}
      </h2>
      <div className="mt-3">{children}</div>
    </section>
  );
}

function PlanList({ title, items }: { title: string; items: { code: string; name: string }[] }) {
  if (items.length === 0) return null;
  return (
    <div className="rounded-md border border-gray-200 bg-white p-3">
      <div className="text-xs font-medium uppercase tracking-wide text-gray-700">{title}</div>
      <ul className="mt-1 space-y-0.5 text-xs">
        {items.map((it, i) => (
          <li key={i}>
            <span className="font-mono text-gray-600">{it.code}</span> — {it.name}
          </li>
        ))}
      </ul>
    </div>
  );
}
