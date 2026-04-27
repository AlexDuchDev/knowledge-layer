"use client";

/**
 * Global recent job runs (Phase 2.1.3).
 *
 * Backed by GET /knowledge-jobs/runs?limit&status&job_type. Operator-only
 * (identity admin). Filters are client-side state; the API call refetches.
 */

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson, formatApiClientError } from "@/lib/api";

type JobRunListing = {
  id: string;
  knowledge_job_id: string;
  job_name: string;
  job_type: string;
  status: string;
  started_at: string;
  completed_at?: string | null;
};

type RunsResponse = {
  items: JobRunListing[];
  count: number;
  limit: number;
  truncated: boolean;
};

const STATUS_OPTIONS = ["", "queued", "running", "completed", "failed", "cancelled"];
const IMPLEMENTED_JOB_TYPES = [
  "",
  "weekly_digest",
  "decision_extraction",
  "planning_summary",
  "stale_scan",
  "support_trends_extraction",
];

export function JobRunsListClient() {
  const [status, setStatus] = useState("");
  const [jobType, setJobType] = useState("");
  const [limit, setLimit] = useState(50);
  const [data, setData] = useState<RunsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ limit: String(limit) });
      if (status) params.set("status", status);
      if (jobType) params.set("job_type", jobType);
      const res = await apiJson<RunsResponse>(`/knowledge-jobs/runs?${params.toString()}`);
      setData(res);
    } catch (err) {
      setError(formatApiClientError(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, jobType, limit]);

  return (
    <div className="mt-6 space-y-4">
      <div className="flex flex-wrap items-end gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <Filter label="Status">
          <select value={status} onChange={(e) => setStatus(e.target.value)} className="rounded-md border border-gray-300 px-2 py-1 text-sm">
            {STATUS_OPTIONS.map((s) => (
              <option key={s} value={s}>{s || "any"}</option>
            ))}
          </select>
        </Filter>
        <Filter label="Job type">
          <select value={jobType} onChange={(e) => setJobType(e.target.value)} className="rounded-md border border-gray-300 px-2 py-1 text-sm">
            {IMPLEMENTED_JOB_TYPES.map((t) => (
              <option key={t} value={t}>{t || "any"}</option>
            ))}
          </select>
        </Filter>
        <Filter label="Limit">
          <select value={limit} onChange={(e) => setLimit(Number(e.target.value))} className="rounded-md border border-gray-300 px-2 py-1 text-sm">
            {[25, 50, 100, 200].map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </Filter>
        <button onClick={load} disabled={loading} className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400">
          {loading ? "Loading…" : "Refresh"}
        </button>
        {data ? <p className="text-xs text-gray-500">{data.count} runs{data.truncated ? " (limit reached)" : ""}</p> : null}
      </div>

      {error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</div>
      ) : null}

      <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white shadow-sm">
        <table className="min-w-full divide-y divide-gray-200 text-sm">
          <thead className="bg-gray-50 text-xs uppercase tracking-wide text-gray-600">
            <tr>
              <Th>Started</Th>
              <Th>Status</Th>
              <Th>Job</Th>
              <Th>Type</Th>
              <Th>Duration</Th>
              <Th>Run</Th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {data?.items?.length ? (
              data.items.map((r) => <Row key={r.id} run={r} />)
            ) : (
              <tr>
                <td colSpan={6} className="px-4 py-6 text-center text-gray-500">
                  {loading ? "Loading…" : "No runs match the current filters."}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function Row({ run }: { run: JobRunListing }) {
  const dur = computeDuration(run.started_at, run.completed_at);
  return (
    <tr className="text-gray-800">
      <Td>{formatTime(run.started_at)}</Td>
      <Td><StatusBadge status={run.status} /></Td>
      <Td>
        <Link href={`/control-plane/jobs/${encodeURIComponent(run.knowledge_job_id)}`} className="text-blue-700 underline hover:text-blue-900">
          {run.job_name || run.knowledge_job_id.slice(0, 8)}
        </Link>
      </Td>
      <Td><span className="font-mono text-xs">{run.job_type}</span></Td>
      <Td>{dur ?? "—"}</Td>
      <Td>
        <Link href={`/control-plane/jobs/runs/${encodeURIComponent(run.id)}`} className="font-mono text-xs text-blue-700 underline hover:text-blue-900">
          {run.id.slice(0, 8)}…
        </Link>
      </Td>
    </tr>
  );
}

function StatusBadge({ status }: { status: string }) {
  const cls =
    status === "completed" ? "bg-green-100 text-green-800" :
    status === "failed" ? "bg-red-100 text-red-800" :
    status === "running" ? "bg-blue-100 text-blue-800" :
    status === "queued" ? "bg-gray-100 text-gray-800" :
    "bg-yellow-100 text-yellow-800";
  return <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${cls}`}>{status}</span>;
}

function Filter({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col text-xs font-medium text-gray-700">
      <span>{label}</span>
      <div className="mt-1">{children}</div>
    </label>
  );
}

function Th({ children }: { children: React.ReactNode }) {
  return <th className="px-4 py-2 text-left">{children}</th>;
}
function Td({ children }: { children: React.ReactNode }) {
  return <td className="px-4 py-2">{children}</td>;
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleString();
  } catch {
    return iso;
  }
}

function computeDuration(startedAt: string, completedAt?: string | null): string | null {
  if (!completedAt) return null;
  try {
    const ms = new Date(completedAt).getTime() - new Date(startedAt).getTime();
    if (ms < 0) return null;
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
    return `${Math.round(ms / 1000)}s`;
  } catch {
    return null;
  }
}
