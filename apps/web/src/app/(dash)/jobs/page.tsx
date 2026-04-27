"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { CatalogPageTemplate } from "@/components/templates/CatalogPageTemplate";
import { WorkflowNextSteps } from "@/components/WorkflowNextSteps";
import { apiJson } from "@/lib/api";

type JobListRow = {
  id: string;
  name: string;
  job_type: string;
  processor_implemented?: boolean;
  template_key?: string | null;
  trigger_type: string;
  status: string;
  processing_mode?: string;
  scenario_binding_count?: number;
  scenario_codes?: string[];
};

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

export default function JobBuilderListPage() {
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [rows, setRows] = useState<JobListRow[] | null>(null);

  const load = useCallback(async () => {
    setErr(null);
    setBusy(true);
    try {
      const data = await apiJson<JobListRow[]>("/knowledge-jobs?expand=scenarios");
      setRows(data);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const description = (
    <span className="block text-sm text-neutral-600">
      Inspect governed job definitions, triggers, and scenario bindings. Badges show whether a job type has a runtime processor; only{" "}
      <code className="rounded bg-neutral-100 px-1">weekly_digest</code>, <code className="rounded bg-neutral-100 px-1">decision_extraction</code>,{" "}
      <code className="rounded bg-neutral-100 px-1">planning_summary</code>, <code className="rounded bg-neutral-100 px-1">stale_scan</code>, and{" "}
      <code className="rounded bg-neutral-100 px-1">support_trends_extraction</code> are runnable locally today.
    </span>
  );

  const actions = (
    <div className="flex gap-3 text-sm">
      <button
        type="button"
        className="rounded-md border border-neutral-300 px-3 py-1.5 disabled:opacity-50"
        disabled={busy}
        onClick={() => void load()}
      >
        Refresh
      </button>
      <Link href="/" className="text-blue-700 underline">
        Home
      </Link>
      <Link href="/control-plane/governance" className="text-blue-700 underline">
        Control plane
      </Link>
    </div>
  );

  return (
    <CatalogPageTemplate title="Job Builder" description={description} actions={actions}>
      <div className="mb-4 rounded-md border border-neutral-200 bg-white px-3 py-2">
        <WorkflowNextSteps variant="operator" />
      </div>

      {err ? <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}

      <section className="overflow-x-auto rounded-lg border border-neutral-200 bg-white shadow-sm">
        <table className="min-w-full text-left text-sm">
          <thead className="border-b border-neutral-200 bg-neutral-50 text-xs font-medium uppercase tracking-wide text-neutral-600">
            <tr>
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Job type / runner</th>
              <th className="px-4 py-3">Template</th>
              <th className="px-4 py-3">Trigger</th>
              <th className="px-4 py-3">Processing</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Scenarios</th>
            </tr>
          </thead>
          <tbody>
            {!rows?.length ? (
              <tr>
                <td colSpan={7} className="px-4 py-8 text-center text-neutral-500">
                  {busy ? "Loading…" : rows ? "No jobs visible for this principal." : "—"}
                </td>
              </tr>
            ) : (
              rows.map((j) => (
                <tr key={j.id} className="border-b border-neutral-100 last:border-0">
                  <td className="px-4 py-3">
                    <Link href={`/control-plane/jobs/${encodeURIComponent(j.id)}`} className="font-medium text-blue-800 hover:underline">
                      {j.name}
                    </Link>
                    <div className="font-mono text-xs text-neutral-500">{j.id}</div>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">
                    <div className="flex flex-wrap items-center gap-2">
                      <span>{j.job_type}</span>
                      {j.processor_implemented === false ? (
                        <span
                          className="rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-sans font-medium uppercase tracking-wide text-amber-900"
                          title="No runtime processor; manual runs will fail until this job_type is implemented."
                        >
                          Not runnable
                        </span>
                      ) : j.processor_implemented === true ? (
                        <span className="rounded bg-emerald-50 px-1.5 py-0.5 text-[10px] font-sans font-medium text-emerald-800">
                          Runnable
                        </span>
                      ) : null}
                    </div>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs">{asStr(j.template_key) || "—"}</td>
                  <td className="px-4 py-3">{j.trigger_type}</td>
                  <td className="px-4 py-3">{asStr(j.processing_mode) || "—"}</td>
                  <td className="px-4 py-3">{j.status}</td>
                  <td className="px-4 py-3 text-xs text-neutral-700">
                    <span className="font-medium">{j.scenario_binding_count ?? 0}</span>
                    {j.scenario_codes?.length ? (
                      <div className="mt-0.5 text-neutral-500">{j.scenario_codes.join(", ")}</div>
                    ) : null}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </section>

      <p className="mt-4 text-xs text-neutral-600">
        <strong>Runnable today:</strong> <code className="rounded bg-neutral-100 px-1">weekly_digest</code>,{" "}
        <code className="rounded bg-neutral-100 px-1">decision_extraction</code>, <code className="rounded bg-neutral-100 px-1">planning_summary</code>,{" "}
        <code className="rounded bg-neutral-100 px-1">stale_scan</code>, and{" "}
        <code className="rounded bg-neutral-100 px-1">support_trends_extraction</code>. Other job types still fail closed with a clear error if executed.
        Use this list to inspect definitions and scenario bindings. Creating or changing jobs is an <strong>advanced</strong> API or builder flow (
        <code className="rounded bg-neutral-100 px-1">POST /knowledge-jobs</code>).
      </p>
    </CatalogPageTemplate>
  );
}
