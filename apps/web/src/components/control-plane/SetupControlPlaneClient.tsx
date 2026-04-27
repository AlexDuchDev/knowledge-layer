"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { apiJson } from "@/lib/api";

type OnboardingTemplate = {
  id: string;
  code: string;
  title: string;
  description: string;
  metadata_json?: {
    connector_families?: string[];
    job_codes?: string[];
    role_codes?: string[];
    scenario_codes?: string[];
  };
};

type SessionSummary = {
  id: string;
  status: string;
  template_code?: string | null;
  updated_at: string;
};

type SessionView = {
  id: string;
  status: string;
  template_code?: string | null;
  selected_presets: Array<{
    preset_type: string;
    preset_code: string;
    slot: string;
  }>;
  connector_selections: Array<{
    connector_family_code: string;
    enabled: boolean;
  }>;
  assignment_drafts?: {
    initial_admin_user_id?: string | null;
  } | null;
};

type LaunchPreview = {
  template_code?: string | null;
  validation_issues: string[];
  planned_roles: PlannedInstantiate[];
  planned_scenarios: PlannedInstantiate[];
  planned_jobs: PlannedInstantiate[];
  connectors_enabled: string[];
};

type PlannedInstantiate = {
  preset_type: string;
  code: string;
  slot: string;
};

type LaunchResult = {
  session_id: string;
  status: string;
  launch_log_id: string;
  created: {
    role_ids: string[];
    scenario_ids: string[];
    job_ids: string[];
  };
};

function Panel({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
      <h2 className="text-sm font-semibold text-neutral-900">{title}</h2>
      <div className="mt-3">{children}</div>
    </section>
  );
}

function ErrorBanner({ error }: { error: string | null }) {
  if (!error) {
    return null;
  }
  return <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{error}</div>;
}

function TemplateSummary({ template }: { template: OnboardingTemplate }) {
  const families = template.metadata_json?.connector_families ?? [];
  const jobs = template.metadata_json?.job_codes ?? [];
  return (
    <div className="rounded-md border border-neutral-200 p-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="font-medium text-neutral-900">{template.title}</div>
          <div className="mt-1 text-sm text-neutral-600">{template.description}</div>
        </div>
        <code className="rounded bg-neutral-100 px-2 py-1 text-xs text-neutral-700">{template.code}</code>
      </div>
      <div className="mt-3 flex flex-wrap gap-2 text-xs text-neutral-600">
        {families.map((family) => (
          <span key={family} className="rounded bg-blue-50 px-2 py-1 text-blue-800">
            {family}
          </span>
        ))}
        {jobs.map((job) => (
          <span key={job} className="rounded bg-emerald-50 px-2 py-1 text-emerald-800">
            {job}
          </span>
        ))}
      </div>
    </div>
  );
}

export function SetupHubPanel() {
  const [templates, setTemplates] = useState<OnboardingTemplate[]>([]);
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const [tpls, sess] = await Promise.all([
        apiJson<OnboardingTemplate[]>("/api/onboarding/templates"),
        apiJson<SessionSummary[]>("/api/onboarding/sessions"),
      ]);
      setTemplates(tpls);
      setSessions(sess);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="space-y-4">
      <ErrorBanner error={error} />
      <Panel title="What setup can launch today">
        <p className="text-sm text-neutral-600">
          This setup flow is now a thin orchestration over real onboarding sessions. It can select seeded templates, preview the actual preset mix,
          and launch instantiated roles, scenarios, and jobs. Connector family picks currently describe supported families for the session:{" "}
          <strong>chat</strong>, <strong>docs/files</strong>, and <strong>calendar/meeting</strong>.
        </p>
        <div className="mt-3 flex gap-3 text-sm">
          <Link href="/control-plane/setup/session/new" className="text-blue-700 underline">
            Start a real setup session
          </Link>
          <Link href="/control-plane/setup/templates" className="text-blue-700 underline">
            Review templates
          </Link>
        </div>
      </Panel>
      <Panel title="Available templates">
        <div className="space-y-3">
          {templates.length ? templates.map((template) => <TemplateSummary key={template.id} template={template} />) : <p className="text-sm text-neutral-500">No setup templates returned.</p>}
        </div>
      </Panel>
      <Panel title="Recent sessions">
        {sessions.length ? (
          <div className="space-y-2">
            {sessions.map((session) => (
              <div key={session.id} className="rounded-md border border-neutral-200 px-3 py-2 text-sm">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div>
                    <div className="font-medium text-neutral-900">{session.template_code || "Untemplated session"}</div>
                    <div className="text-xs text-neutral-500">
                      {session.status} · updated {new Date(session.updated_at).toLocaleString()}
                    </div>
                  </div>
                  <Link href={`/control-plane/setup/session/${encodeURIComponent(session.id)}`} className="text-blue-700 underline">
                    Open session
                  </Link>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-neutral-500">No setup sessions yet.</p>
        )}
      </Panel>
    </div>
  );
}

export function SetupTemplatesPanel() {
  const router = useRouter();
  const [templates, setTemplates] = useState<OnboardingTemplate[]>([]);
  const [busyCode, setBusyCode] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void apiJson<OnboardingTemplate[]>("/api/onboarding/templates")
      .then(setTemplates)
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, []);

  const startSession = useCallback(
    async (templateCode: string) => {
      setBusyCode(templateCode);
      setError(null);
      try {
        const session = await apiJson<SessionView>("/api/onboarding/sessions", { method: "POST", body: "{}" });
        await apiJson<SessionView>(`/api/onboarding/sessions/${encodeURIComponent(session.id)}/select-template`, {
          method: "POST",
          body: JSON.stringify({ template_code: templateCode }),
        });
        router.push(`/control-plane/setup/session/${encodeURIComponent(session.id)}`);
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setBusyCode(null);
      }
    },
    [router],
  );

  return (
    <div className="space-y-4">
      <ErrorBanner error={error} />
      <Panel title="Supported setup templates">
        <p className="text-sm text-neutral-600">
          Choose a template to create a real onboarding session prefilled with supported preset combinations. The resulting session can be previewed and launched.
        </p>
      </Panel>
      {templates.map((template) => (
        <div key={template.id} className="rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <TemplateSummary template={template} />
          <div className="mt-3 flex gap-3 text-sm">
            <button
              type="button"
              className="rounded-md border border-neutral-300 px-3 py-1.5 disabled:opacity-50"
              disabled={busyCode !== null}
              onClick={() => void startSession(template.code)}
            >
              {busyCode === template.code ? "Starting…" : "Start session with this template"}
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

export function NewSetupSessionPanel() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const templateCode = searchParams.get("template");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createSession = useCallback(async () => {
    setBusy(true);
    setError(null);
    try {
      const session = await apiJson<SessionView>("/api/onboarding/sessions", { method: "POST", body: "{}" });
      if (templateCode) {
        await apiJson<SessionView>(`/api/onboarding/sessions/${encodeURIComponent(session.id)}/select-template`, {
          method: "POST",
          body: JSON.stringify({ template_code: templateCode }),
        });
      }
      router.push(`/control-plane/setup/session/${encodeURIComponent(session.id)}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [router, templateCode]);

  return (
    <div className="space-y-4">
      <ErrorBanner error={error} />
      <Panel title="Create onboarding session">
        <p className="text-sm text-neutral-600">
          This creates a persisted setup session you can reopen later. {templateCode ? `Template ${templateCode} will be applied immediately.` : "You can apply a template after creation."}
        </p>
        <div className="mt-3 flex gap-3 text-sm">
          <button type="button" className="rounded-md border border-neutral-300 px-3 py-1.5 disabled:opacity-50" disabled={busy} onClick={() => void createSession()}>
            {busy ? "Creating…" : "Create session"}
          </button>
          <Link href="/control-plane/setup/templates" className="text-blue-700 underline">
            Pick a template first
          </Link>
        </div>
      </Panel>
    </div>
  );
}

export function SetupSessionPanel({ sessionId }: { sessionId: string }) {
  const [session, setSession] = useState<SessionView | null>(null);
  const [preview, setPreview] = useState<LaunchPreview | null>(null);
  const [launchResult, setLaunchResult] = useState<LaunchResult | null>(null);
  const [busyAction, setBusyAction] = useState<"refresh" | "preview" | "launch" | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setBusyAction("refresh");
    setError(null);
    try {
      const sess = await apiJson<SessionView>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}`);
      setSession(sess);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusyAction(null);
    }
  }, [sessionId]);

  useEffect(() => {
    void load();
  }, [load]);

  const runPreview = useCallback(async () => {
    setBusyAction("preview");
    setError(null);
    try {
      const result = await apiJson<LaunchPreview>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}/preview`, {
        method: "POST",
        body: "{}",
      });
      setPreview(result);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusyAction(null);
    }
  }, [sessionId]);

  const launch = useCallback(async () => {
    setBusyAction("launch");
    setError(null);
    try {
      const result = await apiJson<LaunchResult>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}/launch`, {
        method: "POST",
        body: "{}",
      });
      setLaunchResult(result);
      const refreshed = await apiJson<SessionView>(`/api/onboarding/sessions/${encodeURIComponent(sessionId)}`);
      setSession(refreshed);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusyAction(null);
    }
  }, [sessionId]);

  const enabledFamilies = useMemo(
    () => (session?.connector_selections ?? []).filter((row) => row.enabled).map((row) => row.connector_family_code),
    [session],
  );

  return (
    <div className="space-y-4">
      <ErrorBanner error={error} />
      <Panel title="Session state">
        {session ? (
          <div className="space-y-3 text-sm">
            <div>
              <div className="font-medium text-neutral-900">{session.template_code || "Untemplated session"}</div>
              <div className="text-neutral-600">
                Status: <strong>{session.status}</strong>
              </div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wide text-neutral-500">Selected presets</div>
              <div className="mt-1 flex flex-wrap gap-2">
                {session.selected_presets.length ? (
                  session.selected_presets.map((preset) => (
                    <span key={`${preset.slot}-${preset.preset_code}`} className="rounded bg-neutral-100 px-2 py-1 text-xs text-neutral-700">
                      {preset.preset_type}:{preset.preset_code}
                    </span>
                  ))
                ) : (
                  <span className="text-neutral-500">No presets selected.</span>
                )}
              </div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wide text-neutral-500">Enabled connector families</div>
              <div className="mt-1 flex flex-wrap gap-2">
                {enabledFamilies.length ? enabledFamilies.map((family) => <span key={family} className="rounded bg-blue-50 px-2 py-1 text-xs text-blue-800">{family}</span>) : <span className="text-neutral-500">No families enabled.</span>}
              </div>
            </div>
            <div className="flex gap-3">
              <button type="button" className="rounded-md border border-neutral-300 px-3 py-1.5 disabled:opacity-50" disabled={busyAction !== null} onClick={() => void load()}>
                {busyAction === "refresh" ? "Refreshing…" : "Refresh session"}
              </button>
              <button type="button" className="rounded-md border border-neutral-300 px-3 py-1.5 disabled:opacity-50" disabled={busyAction !== null} onClick={() => void runPreview()}>
                {busyAction === "preview" ? "Previewing…" : "Run launch preview"}
              </button>
              <button type="button" className="rounded-md border border-neutral-300 px-3 py-1.5 disabled:opacity-50" disabled={busyAction !== null} onClick={() => void launch()}>
                {busyAction === "launch" ? "Launching…" : "Launch"}
              </button>
            </div>
          </div>
        ) : (
          <p className="text-sm text-neutral-500">Loading session…</p>
        )}
      </Panel>

      <Panel title="Launch preview">
        {preview ? (
          <div className="space-y-3 text-sm">
            {preview.validation_issues.length ? (
              <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-amber-900">
                {preview.validation_issues.map((issue) => (
                  <div key={issue}>{issue}</div>
                ))}
              </div>
            ) : (
              <div className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-emerald-900">Preview passed. This session is launchable.</div>
            )}
            <div className="grid gap-3 md:grid-cols-3">
              <PreviewList title="Roles" rows={preview.planned_roles} />
              <PreviewList title="Scenarios" rows={preview.planned_scenarios} />
              <PreviewList title="Jobs" rows={preview.planned_jobs} />
            </div>
          </div>
        ) : (
          <p className="text-sm text-neutral-500">Run preview to inspect the actual instantiated objects before launch.</p>
        )}
      </Panel>

      <Panel title="Launch result">
        {launchResult ? (
          <div className="space-y-2 text-sm text-neutral-700">
            <div>Session status: <strong>{launchResult.status}</strong></div>
            <div>Created roles: {launchResult.created.role_ids.length}</div>
            <div>Created scenarios: {launchResult.created.scenario_ids.length}</div>
            <div>Created jobs: {launchResult.created.job_ids.length}</div>
            <div className="text-xs text-neutral-500">Launch log: {launchResult.launch_log_id}</div>
          </div>
        ) : (
          <p className="text-sm text-neutral-500">Launch from this page to create the selected presets and record the result.</p>
        )}
      </Panel>
    </div>
  );
}

function PreviewList({ title, rows }: { title: string; rows: PlannedInstantiate[] }) {
  return (
    <div className="rounded-md border border-neutral-200 p-3">
      <div className="text-sm font-medium text-neutral-900">{title}</div>
      <div className="mt-2 space-y-1 text-xs text-neutral-700">
        {rows.length ? rows.map((row) => <div key={`${row.slot}-${row.code}`}>{row.code}</div>) : <div className="text-neutral-500">None selected.</div>}
      </div>
    </div>
  );
}

export function SetupPreviewPanel() {
  const searchParams = useSearchParams();
  const sessionId = searchParams.get("session");
  if (!sessionId) {
    return <Panel title="Launch preview">Add `?session=&lt;session-id&gt;` to preview a real onboarding session.</Panel>;
  }
  return <SetupSessionPanel sessionId={sessionId} />;
}

export function SetupLaunchResultPanel() {
  const searchParams = useSearchParams();
  const sessionId = searchParams.get("session");
  if (!sessionId) {
    return <Panel title="Launch result">Add `?session=&lt;session-id&gt;` to inspect a real session and launch result.</Panel>;
  }
  return <SetupSessionPanel sessionId={sessionId} />;
}
