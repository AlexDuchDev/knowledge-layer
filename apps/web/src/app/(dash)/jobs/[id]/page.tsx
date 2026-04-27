"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { FieldHint } from "@/components/guidance/FieldHint";
import { apiBase, apiJson, isDevPrincipalHeader, principalUserId } from "@/lib/api";

type KnowledgeJob = {
  id: string;
  name: string;
  job_type: string;
  purpose?: string | null;
  description?: string | null;
  owner_id: string;
  source_scope_json: unknown;
  operator_scope_json: unknown;
  trigger_type: string;
  output_domain_id?: string | null;
  output_sensitivity_level: number;
  publication_mode: string;
  review_required: boolean;
  citations_required: boolean;
  provenance_required: boolean;
  scenario_only_exposure: boolean;
  allow_domain_run_job: boolean;
  template_key?: string | null;
  processing_mode: string;
  cloned_from_job_id?: string | null;
  config_json: unknown;
  status: string;
};

type JobTrigger = {
  id: string;
  trigger_type: string;
  schedule_expr?: string | null;
  status: string;
};

type OperatorRow = { principal_type: string; principal_id: string };

type JobPreview = Record<string, unknown>;

type EngineMetadata = {
  implemented_job_types: string[];
};

const PROCESSING_MODES = ["summarize", "extract", "consolidate", "detect", "transform", "publish"] as const;

const PUBLICATION_MODES = ["draft_only", "reviewed_publish", "auto_publish"] as const;

function stringifyJSON(v: unknown, fallback = "{}"): string {
  try {
    return JSON.stringify(v ?? {}, null, 2);
  } catch {
    return fallback;
  }
}

function parseJSONObject(text: string): object {
  const v = JSON.parse(text) as unknown;
  if (v === null || typeof v !== "object" || Array.isArray(v)) {
    throw new Error("expected a JSON object");
  }
  return v as object;
}

export default function JobBuilderDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = typeof params.id === "string" ? params.id : "";

  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [job, setJob] = useState<KnowledgeJob | null>(null);
  const [triggers, setTriggers] = useState<JobTrigger[]>([]);
  const [operators, setOperators] = useState<OperatorRow[]>([]);
  const [preview, setPreview] = useState<JobPreview | null>(null);
  const [dryResult, setDryResult] = useState<{ valid: boolean; preview: JobPreview } | null>(null);

  const [name, setName] = useState("");
  const [status, setStatus] = useState("draft");
  const [purpose, setPurpose] = useState("");
  const [description, setDescription] = useState("");
  const [sourceScopeText, setSourceScopeText] = useState("{}");
  const [operatorScopeText, setOperatorScopeText] = useState("{}");
  const [configText, setConfigText] = useState("{}");
  const [triggerType, setTriggerType] = useState("manual");
  const [processingMode, setProcessingMode] = useState("summarize");
  const [publicationMode, setPublicationMode] = useState("draft_only");
  const [outputDomainId, setOutputDomainId] = useState("");
  const [outputSensitivity, setOutputSensitivity] = useState(0);
  const [reviewRequired, setReviewRequired] = useState(false);
  const [citationsRequired, setCitationsRequired] = useState(false);
  const [provenanceRequired, setProvenanceRequired] = useState(true);
  const [scenarioOnly, setScenarioOnly] = useState(false);
  const [allowDomainRun, setAllowDomainRun] = useState(true);
  const [operatorUserIdsText, setOperatorUserIdsText] = useState("");
  const [scenarioBindingsText, setScenarioBindingsText] = useState("[]");
  const [newTrigType, setNewTrigType] = useState("schedule");
  const [newSched, setNewSched] = useState("0 9 * * 1");
  const [engineMeta, setEngineMeta] = useState<EngineMetadata | null>(null);

  const applyJobToForm = useCallback((j: KnowledgeJob) => {
    setName(j.name);
    setStatus(j.status);
    setPurpose(j.purpose ?? "");
    setDescription(j.description ?? "");
    setSourceScopeText(stringifyJSON(j.source_scope_json));
    setOperatorScopeText(stringifyJSON(j.operator_scope_json));
    setConfigText(stringifyJSON(j.config_json));
    setTriggerType(j.trigger_type);
    setProcessingMode(j.processing_mode || "summarize");
    setPublicationMode(j.publication_mode);
    setOutputDomainId(j.output_domain_id ?? "");
    setOutputSensitivity(j.output_sensitivity_level);
    setReviewRequired(j.review_required);
    setCitationsRequired(j.citations_required);
    setProvenanceRequired(j.provenance_required);
    setScenarioOnly(j.scenario_only_exposure);
    setAllowDomainRun(j.allow_domain_run_job);
  }, []);

  const reload = useCallback(async () => {
    if (!id) return;
    setErr(null);
    setBusy(true);
    try {
      const [j, tr, op] = await Promise.all([
        apiJson<KnowledgeJob>(`/knowledge-jobs/${encodeURIComponent(id)}`),
        apiJson<JobTrigger[]>(`/knowledge-jobs/${encodeURIComponent(id)}/triggers`),
        apiJson<OperatorRow[]>(`/knowledge-jobs/${encodeURIComponent(id)}/operators`),
      ]);
      setJob(j);
      applyJobToForm(j);
      setTriggers(tr);
      setOperators(op);
      const users = op.filter((o) => o.principal_type === "user").map((o) => o.principal_id);
      setOperatorUserIdsText(users.join(", "));
      const pv = await apiJson<JobPreview>(`/knowledge-jobs/${encodeURIComponent(id)}/preview`);
      const bindings = (pv.scenario_bindings as { scenario_id: string; relationship: string }[] | undefined) ?? [];
      setScenarioBindingsText(JSON.stringify(bindings.map((b) => ({ scenario_id: b.scenario_id, relationship: b.relationship })), null, 2));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [id, applyJobToForm]);

  useEffect(() => {
    void reload();
  }, [reload]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const meta = await apiJson<EngineMetadata>("/knowledge-jobs/engine-metadata");
        if (!cancelled) setEngineMeta(meta);
      } catch {
        if (!cancelled) setEngineMeta(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const run = useCallback(async (fn: () => Promise<void>) => {
    setErr(null);
    setBusy(true);
    try {
      await fn();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, []);

  const patch = useCallback(
    async (body: Record<string, unknown>) => {
      await run(async () => {
        const j = await apiJson<KnowledgeJob>(`/knowledge-jobs/${encodeURIComponent(id)}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
        setJob(j);
        applyJobToForm(j);
      });
    },
    [id, run, applyJobToForm],
  );

  const nav = useMemo(
    () =>
      [
        { href: "#basic", label: "Basic" },
        { href: "#scope", label: "Source scope" },
        { href: "#triggers", label: "Triggers" },
        { href: "#processing", label: "Processing" },
        { href: "#output", label: "Output" },
        { href: "#governance", label: "Governance" },
        { href: "#operators", label: "Operators" },
        { href: "#scenarios", label: "Scenarios" },
        { href: "#preview", label: "Preview" },
        { href: "#run", label: "Test run" },
      ] as const,
    [],
  );

  const processorRunnable = useMemo(() => {
    if (!job || !engineMeta?.implemented_job_types?.length) return null;
    const jt = job.job_type.trim().toLowerCase();
    return engineMeta.implemented_job_types.some((x) => x.trim().toLowerCase() === jt);
  }, [job, engineMeta]);

  if (!id) {
    return <main className="p-6 text-sm text-red-700">Missing job id.</main>;
  }

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link href="/control-plane/jobs" className="text-sm text-blue-700 underline">
            ← Job Builder
          </Link>
          <h1 className="mt-2 text-2xl font-semibold tracking-tight">{job?.name ?? "Job"}</h1>
          <p className="mt-1 font-mono text-xs text-neutral-500">{id}</p>
          <p className="mt-1 text-sm text-neutral-600">
            API <code className="rounded bg-neutral-100 px-1">{apiBase()}</code>
            {isDevPrincipalHeader() ? (
              <>
                {" "}
                · <code className="rounded bg-neutral-100 px-1">{principalUserId()}</code>
              </>
            ) : null}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
            disabled={busy}
            onClick={() => void reload()}
          >
            Reload
          </button>
          <button
            type="button"
            className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() =>
              run(async () => {
                const j = await apiJson<KnowledgeJob>(`/knowledge-jobs/${encodeURIComponent(id)}/clone`, { method: "POST" });
                router.push(`/control-plane/jobs/${encodeURIComponent(j.id)}`);
              })
            }
          >
            Clone
          </button>
        </div>
      </div>

      <nav className="mb-8 flex flex-wrap gap-2 border-b border-neutral-200 pb-3 text-sm">
        {nav.map((n) => (
          <a key={n.href} href={n.href} className="rounded-full bg-neutral-100 px-3 py-1 text-neutral-800 hover:bg-neutral-200">
            {n.label}
          </a>
        ))}
      </nav>

      {err ? <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}

      {job && engineMeta && processorRunnable === false ? (
        <div className="mb-4 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950">
          <strong>Not runnable:</strong> job type <span className="font-mono">{job.job_type}</span> has no runtime processor in this build. Queued runs will{" "}
          <strong>fail closed</strong>. Supported types:{" "}
          <span className="font-mono">{engineMeta.implemented_job_types.join(", ")}</span>. See{" "}
          <code className="rounded bg-amber-100/80 px-1">GET /knowledge-jobs/engine-metadata</code> and repo docs{" "}
          <span className="font-mono">LIMITATIONS.md</span> / <span className="font-mono">OSS_V1_SCOPE.md</span>.
        </div>
      ) : null}

      <div className="space-y-10">
        <section id="basic" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Basic</h2>
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <label className="text-sm">
              <span className="text-neutral-600">Name</span>
              <input className="mt-1 w-full rounded border px-2 py-1.5 text-sm" value={name} onChange={(e) => setName(e.target.value)} />
            </label>
            <label className="text-sm">
              <span className="text-neutral-600">Status</span>
              <select className="mt-1 w-full rounded border px-2 py-1.5 text-sm" value={status} onChange={(e) => setStatus(e.target.value)}>
                <option value="draft">draft</option>
                <option value="active">active</option>
                <option value="archived">archived</option>
              </select>
            </label>
            <label className="text-sm sm:col-span-2">
              <span className="text-neutral-600">Purpose</span>
              <input className="mt-1 w-full rounded border px-2 py-1.5 text-sm" value={purpose} onChange={(e) => setPurpose(e.target.value)} />
            </label>
            <label className="text-sm sm:col-span-2">
              <span className="text-neutral-600">Description</span>
              <textarea className="mt-1 w-full rounded border px-2 py-1.5 text-sm" rows={2} value={description} onChange={(e) => setDescription(e.target.value)} />
            </label>
          </div>
          <button
            type="button"
            className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() =>
              patch({
                name: name.trim(),
                status: status.trim(),
                purpose: purpose.trim() || null,
                description: description.trim() || null,
              })
            }
          >
            Save basic
          </button>
          {job ? (
            <p className="mt-2 text-xs text-neutral-500">
              Type <span className="font-mono">{job.job_type}</span>
              {job.template_key ? (
                <>
                  {" "}
                  · template <span className="font-mono">{job.template_key}</span>
                </>
              ) : null}
              {job.cloned_from_job_id ? (
                <>
                  {" "}
                  · cloned from{" "}
                  <Link href={`/control-plane/jobs/${job.cloned_from_job_id}`} className="font-mono text-blue-700 underline">
                    {job.cloned_from_job_id}
                  </Link>
                </>
              ) : null}
            </p>
          ) : null}
        </section>

        <section id="scope" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Source scope & config</h2>
          <p className="mt-1 text-xs text-neutral-500">JSON objects. Source scope is validated against job type; changing it resyncs knowledge_job_sources when possible.</p>
          <label className="mt-3 block text-sm">
            <span className="text-neutral-600">source_scope_json</span>
            <textarea className="mt-1 w-full rounded border px-2 py-1.5 font-mono text-xs" rows={8} value={sourceScopeText} onChange={(e) => setSourceScopeText(e.target.value)} />
          </label>
          <label className="mt-3 block text-sm">
            <span className="text-neutral-600">operator_scope_json</span>
            <textarea className="mt-1 w-full rounded border px-2 py-1.5 font-mono text-xs" rows={4} value={operatorScopeText} onChange={(e) => setOperatorScopeText(e.target.value)} />
          </label>
          <label className="mt-3 block text-sm">
            <span className="text-neutral-600">config_json</span>
            <textarea className="mt-1 w-full rounded border px-2 py-1.5 font-mono text-xs" rows={6} value={configText} onChange={(e) => setConfigText(e.target.value)} />
          </label>
          <button
            type="button"
            className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() => {
              try {
                const source_scope_json = parseJSONObject(sourceScopeText);
                const operator_scope_json = parseJSONObject(operatorScopeText);
                const config_json = parseJSONObject(configText);
                void patch({ source_scope_json, operator_scope_json, config_json });
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              }
            }}
          >
            Save scope & config
          </button>
        </section>

        <section id="triggers" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Triggers & primary type</h2>
          <FieldHint>
            trigger_type names how this job is invoked (e.g. schedule-driven vs manual). It must match an active trigger row unless the job is manual-only.
          </FieldHint>
          <label className="mt-3 block max-w-xs text-sm">
            Primary trigger_type
            <input className="mt-1 w-full rounded border px-2 py-1.5 font-mono text-sm" value={triggerType} onChange={(e) => setTriggerType(e.target.value)} />
          </label>
          <button
            type="button"
            className="mt-2 rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
            disabled={busy}
            onClick={() => void patch({ trigger_type: triggerType.trim() })}
          >
            Save primary trigger type
          </button>
          <ul className="mt-4 space-y-2 text-sm">
            {triggers.length === 0 ? <li className="text-neutral-500">No triggers.</li> : null}
            {triggers.map((t) => (
              <li key={t.id} className="rounded border border-neutral-100 bg-neutral-50 px-3 py-2 font-mono text-xs">
                {t.trigger_type} · {t.status}
                {t.schedule_expr ? ` · ${t.schedule_expr}` : ""} · {t.id}
              </li>
            ))}
          </ul>
          <div className="mt-4 flex flex-wrap gap-2">
            <input className="rounded border px-2 py-1 text-sm" value={newTrigType} onChange={(e) => setNewTrigType(e.target.value)} placeholder="trigger_type" />
            <input className="rounded border px-2 py-1 text-sm" value={newSched} onChange={(e) => setNewSched(e.target.value)} placeholder="schedule_expr" />
            <button
              type="button"
              className="rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  await apiJson(`/knowledge-jobs/${encodeURIComponent(id)}/triggers`, {
                    method: "POST",
                    body: JSON.stringify({ trigger_type: newTrigType.trim(), schedule_expr: newSched.trim() }),
                  });
                  const tr = await apiJson<JobTrigger[]>(`/knowledge-jobs/${encodeURIComponent(id)}/triggers`);
                  setTriggers(tr);
                })
              }
            >
              Add trigger
            </button>
          </div>
        </section>

        <section id="processing" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Processing mode</h2>
          <select className="mt-2 rounded border px-2 py-1.5 text-sm" value={processingMode} onChange={(e) => setProcessingMode(e.target.value)}>
            {PROCESSING_MODES.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
          <button type="button" className="ml-2 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50" disabled={busy} onClick={() => void patch({ processing_mode: processingMode })}>
            Save
          </button>
        </section>

        <section id="output" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Output policy</h2>
          <FieldHint>
            publication_mode and output_sensitivity_level govern how job outputs are stored and who can see them; align with governance rules in your domain.
          </FieldHint>
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <label className="text-sm sm:col-span-2">
              publication_mode
              <select className="mt-1 w-full rounded border px-2 py-1.5 text-sm" value={publicationMode} onChange={(e) => setPublicationMode(e.target.value)}>
                {PUBLICATION_MODES.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            </label>
            <label className="text-sm">
              output_domain_id (UUID)
              <input className="mt-1 w-full rounded border px-2 py-1.5 font-mono text-xs" value={outputDomainId} onChange={(e) => setOutputDomainId(e.target.value)} />
            </label>
            <label className="text-sm">
              output_sensitivity_level
              <input
                type="number"
                className="mt-1 w-full rounded border px-2 py-1.5 text-sm"
                value={outputSensitivity}
                onChange={(e) => setOutputSensitivity(Number(e.target.value))}
              />
              <FieldHint>Numeric sensitivity for produced artifacts; higher values restrict visibility.</FieldHint>
            </label>
            <label className="flex items-center gap-2 text-sm sm:col-span-2">
              <input type="checkbox" checked={reviewRequired} onChange={(e) => setReviewRequired(e.target.checked)} />
              review_required
            </label>
          </div>
          <button
            type="button"
            className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() => {
              const body: Record<string, unknown> = {
                publication_mode: publicationMode,
                output_sensitivity_level: outputSensitivity,
                review_required: reviewRequired,
              };
              const trimmed = outputDomainId.trim();
              body.output_domain_id = trimmed === "" ? null : trimmed;
              void patch(body);
            }}
          >
            Save output policy
          </button>
        </section>

        <section id="governance" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Governance flags</h2>
          <div className="mt-3 space-y-2 text-sm">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={citationsRequired} onChange={(e) => setCitationsRequired(e.target.checked)} />
              citations_required
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={provenanceRequired} onChange={(e) => setProvenanceRequired(e.target.checked)} />
              provenance_required
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={scenarioOnly} onChange={(e) => setScenarioOnly(e.target.checked)} />
              scenario_only_exposure
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={allowDomainRun} onChange={(e) => setAllowDomainRun(e.target.checked)} />
              allow_domain_run_job (when off, only owner and explicit operators may run)
            </label>
          </div>
          <button
            type="button"
            className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() =>
              void patch({
                citations_required: citationsRequired,
                provenance_required: provenanceRequired,
                scenario_only_exposure: scenarioOnly,
                allow_domain_run_job: allowDomainRun,
              })
            }
          >
            Save governance
          </button>
        </section>

        <section id="operators" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Operators</h2>
          <p className="mt-1 text-xs text-neutral-500">Comma-separated user UUIDs → replaces knowledge_job_operators (principal_type user).</p>
          <textarea className="mt-2 w-full rounded border px-2 py-1.5 font-mono text-xs" rows={2} value={operatorUserIdsText} onChange={(e) => setOperatorUserIdsText(e.target.value)} />
          <button
            type="button"
            className="mt-2 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() =>
              run(async () => {
                const parts = operatorUserIdsText
                  .split(/[\s,]+/)
                  .map((s) => s.trim())
                  .filter(Boolean);
                await apiJson(`/knowledge-jobs/${encodeURIComponent(id)}/operators`, {
                  method: "POST",
                  body: JSON.stringify({ user_ids: parts }),
                });
                const op = await apiJson<OperatorRow[]>(`/knowledge-jobs/${encodeURIComponent(id)}/operators`);
                setOperators(op);
              })
            }
          >
            Replace operators
          </button>
          <ul className="mt-3 text-xs text-neutral-600">
            {operators.map((o) => (
              <li key={`${o.principal_type}-${o.principal_id}`}>
                {o.principal_type}: {o.principal_id}
              </li>
            ))}
          </ul>
        </section>

        <section id="scenarios" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Scenario bindings</h2>
          <p className="mt-1 text-xs text-neutral-500">
            JSON array of <code className="rounded bg-neutral-100 px-1">{"{ scenario_id, relationship }"}</code>. Relationship: primary_support | supports | optional. Body is a raw array on{" "}
            <code className="rounded bg-neutral-100 px-1">POST /knowledge-jobs/:id/scenario-bindings</code>.
          </p>
          <textarea className="mt-2 w-full rounded border px-2 py-1.5 font-mono text-xs" rows={6} value={scenarioBindingsText} onChange={(e) => setScenarioBindingsText(e.target.value)} />
          <button
            type="button"
            className="mt-2 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() =>
              run(async () => {
                const parsed = JSON.parse(scenarioBindingsText) as unknown;
                if (!Array.isArray(parsed)) throw new Error("expected JSON array");
                await apiJson(`/knowledge-jobs/${encodeURIComponent(id)}/scenario-bindings`, {
                  method: "POST",
                  body: JSON.stringify(parsed),
                });
                await reload();
              })
            }
          >
            Save scenario bindings
          </button>
        </section>

        <section id="preview" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Preview</h2>
          <button
            type="button"
            className="mt-2 rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
            disabled={busy}
            onClick={() =>
              run(async () => {
                const p = await apiJson<JobPreview>(`/knowledge-jobs/${encodeURIComponent(id)}/preview`);
                setPreview(p);
              })
            }
          >
            Load preview
          </button>
          {preview ? <pre className="mt-3 max-h-96 overflow-auto rounded bg-neutral-50 p-3 text-xs">{JSON.stringify(preview, null, 2)}</pre> : null}
        </section>

        <section id="run" className="scroll-mt-24 rounded-lg border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-lg font-medium">Test run</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            <button
              type="button"
              className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  const r = await apiJson<{ valid: boolean; preview: JobPreview }>(`/knowledge-jobs/${encodeURIComponent(id)}/test-run`, {
                    method: "POST",
                    body: JSON.stringify({ dry_run: true }),
                  });
                  setDryResult(r);
                  setPreview(r.preview);
                })
              }
            >
              Dry run (validate + preview)
            </button>
            <button
              type="button"
              className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  await apiJson(`/knowledge-jobs/${encodeURIComponent(id)}/test-run`, {
                    method: "POST",
                    body: JSON.stringify({ dry_run: false }),
                  });
                })
              }
            >
              Enqueue run
            </button>
            <Link href={`/control-plane/jobs`} className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm leading-6 text-blue-800">
              List runs from job list (debug)
            </Link>
          </div>
          {dryResult ? (
            <p className="mt-2 text-sm">
              Valid: <strong>{dryResult.valid ? "yes" : "no"}</strong>
            </p>
          ) : null}
        </section>
      </div>
    </main>
  );
}
