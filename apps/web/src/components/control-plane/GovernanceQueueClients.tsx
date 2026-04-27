"use client";

/**
 * Native CP governance queue clients (Phase 2.1.x — replaces six stubs that
 * previously linked to product `/governance/*` pages).
 *
 * Each queue is a thin operator-context view over the existing API:
 *   - ReviewsQueueClient        → GET /review-tasks
 *   - ApprovalsQueueClient      → GET /governance/approval-queue
 *   - StaleContentQueueClient   → GET /governance/stale-content
 *   - FailedJobsQueueClient     → GET /ops/failed-runs (job_runs)
 *   - FailedSyncsQueueClient    → GET /ops/failed-runs (ingestion_runs)
 *   - PolicyExceptionsQueueClient → GET /governance/policy-exceptions
 *
 * They share a small `QueueShell` for load/refresh/error UX. Bulk-action
 * affordances (approve / dismiss / retry) are intentionally NOT auto-wired
 * here — actions live in dedicated edit flows and the existing review/approve
 * mutation routes (POST /review-tasks/:id/..., POST /governance/...) — to keep
 * the queue read-only at this iteration. Adding bulk actions later only
 * requires extending each client; the API surface is already there.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { apiJson, formatApiClientError } from "@/lib/api";

type Json = Record<string, unknown>;

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

function shortId(id: string): string {
  return id ? `${id.slice(0, 8)}…` : "—";
}

function fmtTime(v: unknown): string {
  if (typeof v !== "string" || !v) return "—";
  try {
    return new Date(v).toLocaleString();
  } catch {
    return v;
  }
}

function QueueShell<T>({
  load,
  rows,
  err,
  busy,
  emptyMessage,
  renderRow,
  countLabel,
}: {
  load: () => void;
  rows: T[] | null;
  err: string | null;
  busy: boolean;
  emptyMessage: string;
  renderRow: (row: T, i: number) => React.ReactNode;
  countLabel: (n: number) => string;
}) {
  return (
    <div className="mt-6 space-y-4">
      <div className="flex items-center gap-3">
        <button
          onClick={load}
          disabled={busy}
          className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400"
        >
          {busy ? "Loading…" : rows == null ? "Load queue" : "Refresh"}
        </button>
        {rows != null ? <span className="text-xs text-gray-600">{countLabel(rows.length)}</span> : null}
      </div>
      {err ? <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{err}</div> : null}
      {rows && rows.length === 0 ? (
        <p className="rounded-md border border-gray-200 bg-white p-6 text-sm text-gray-600">{emptyMessage}</p>
      ) : null}
      {rows && rows.length > 0 ? (
        <ul className="divide-y divide-gray-100 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
          {rows.map(renderRow)}
        </ul>
      ) : null}
    </div>
  );
}

function useQueue<T>(load: () => Promise<T[]>) {
  const [rows, setRows] = useState<T[] | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const refresh = useCallback(async () => {
    setBusy(true);
    setErr(null);
    try {
      setRows(await load());
    } catch (e) {
      setErr(formatApiClientError(e));
      setRows(null);
    } finally {
      setBusy(false);
    }
  }, [load]);
  useEffect(() => {
    void refresh();
  }, [refresh]);
  return { rows, err, busy, refresh };
}

function StatusBadge({ status }: { status: string }) {
  const cls =
    status === "open" || status === "pending" || status === "queued"
      ? "bg-yellow-100 text-yellow-800"
      : status === "approved" || status === "completed"
      ? "bg-green-100 text-green-800"
      : status === "failed" || status === "rejected"
      ? "bg-red-100 text-red-800"
      : "bg-gray-100 text-gray-800";
  return <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${cls}`}>{status}</span>;
}

// ----- Reviews ---------------------------------------------------------

type ReviewTask = {
  id: string;
  target_type: string;
  target_id: string;
  status: string;
  created_at: string;
};

export function ReviewsQueueClient() {
  const load = useCallback(() => apiJson<ReviewTask[]>("/review-tasks"), []);
  const { rows, err, busy, refresh } = useQueue(load);
  return (
    <QueueShell<ReviewTask>
      load={refresh}
      rows={rows}
      err={err}
      busy={busy}
      emptyMessage="No open review tasks."
      countLabel={(n) => `${n} open task(s)`}
      renderRow={(t) => (
        <li key={t.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
          <div>
            <StatusBadge status={t.status} />
            <span className="ml-2 text-gray-700">
              {t.target_type} · <span className="font-mono text-xs">{shortId(t.target_id)}</span>
            </span>
            <span className="ml-2 text-xs text-gray-500">created {fmtTime(t.created_at)}</span>
          </div>
          {t.target_type === "entity" ? (
            <Link href={`/entities/${encodeURIComponent(t.target_id)}`} className="text-sm text-blue-700 underline hover:text-blue-900">
              Open entity
            </Link>
          ) : null}
        </li>
      )}
    />
  );
}

// ----- Approvals -------------------------------------------------------

export function ApprovalsQueueClient() {
  const load = useCallback(() => apiJson<Json[]>("/governance/approval-queue"), []);
  const { rows, err, busy, refresh } = useQueue(load);
  return (
    <QueueShell<Json>
      load={refresh}
      rows={rows}
      err={err}
      busy={busy}
      emptyMessage="No pending approvals."
      countLabel={(n) => `${n} item(s) awaiting publish approval`}
      renderRow={(r, i) => {
        const id = asStr(r.id);
        const targetType = asStr(r.target_type);
        const targetID = asStr(r.target_id);
        const status = asStr(r.status) || "pending";
        return (
          <li key={id || i} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
            <div>
              <StatusBadge status={status} />
              <span className="ml-2 text-gray-700">
                {targetType || "—"} · <span className="font-mono text-xs">{shortId(targetID)}</span>
              </span>
            </div>
            {targetType === "entity" && targetID ? (
              <Link href={`/entities/${encodeURIComponent(targetID)}`} className="text-sm text-blue-700 underline hover:text-blue-900">
                Open entity
              </Link>
            ) : null}
          </li>
        );
      }}
    />
  );
}

// ----- Stale content ---------------------------------------------------

export function StaleContentQueueClient() {
  const load = useCallback(() => apiJson<Json[]>("/governance/stale-content"), []);
  const { rows, err, busy, refresh } = useQueue(load);
  return (
    <QueueShell<Json>
      load={refresh}
      rows={rows}
      err={err}
      busy={busy}
      emptyMessage="No entities flagged stale."
      countLabel={(n) => `${n} stale entit${n === 1 ? "y" : "ies"}`}
      renderRow={(r, i) => {
        const id = asStr(r.entity_id || r.id);
        const title = asStr(r.title) || "—";
        const lastUpdated = asStr(r.updated_at || r.last_updated);
        return (
          <li key={id || i} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
            <div>
              <span className="font-medium text-gray-900">{title}</span>
              <span className="ml-2 text-xs text-gray-500">
                <span className="font-mono">{shortId(id)}</span> · last updated {fmtTime(lastUpdated)}
              </span>
            </div>
            {id ? (
              <Link href={`/entities/${encodeURIComponent(id)}`} className="text-sm text-blue-700 underline hover:text-blue-900">
                Open entity
              </Link>
            ) : null}
          </li>
        );
      }}
    />
  );
}

// ----- Failed runs (jobs / syncs share /ops/failed-runs) ---------------

type FailedRunsResponse = {
  ingestion_runs: Json[];
  job_runs: Json[];
};

function useFailedRuns(slice: "ingestion_runs" | "job_runs") {
  const load = useCallback(async () => {
    const data = await apiJson<FailedRunsResponse>("/ops/failed-runs");
    return (data?.[slice] ?? []) as Json[];
  }, [slice]);
  return useQueue(load);
}

export function FailedJobsQueueClient() {
  const { rows, err, busy, refresh } = useFailedRuns("job_runs");
  return (
    <QueueShell<Json>
      load={refresh}
      rows={rows}
      err={err}
      busy={busy}
      emptyMessage="No recent failed job runs."
      countLabel={(n) => `${n} recent failed job run(s)`}
      renderRow={(r, i) => {
        const runID = asStr(r.id);
        const jobID = asStr(r.knowledge_job_id);
        const startedAt = asStr(r.started_at);
        return (
          <li key={runID || i} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
            <div>
              <StatusBadge status={asStr(r.status) || "failed"} />
              <span className="ml-2 text-gray-700">
                Job <span className="font-mono text-xs">{shortId(jobID)}</span>
              </span>
              <span className="ml-2 text-xs text-gray-500">started {fmtTime(startedAt)}</span>
            </div>
            <div className="flex gap-3 text-sm">
              {jobID ? (
                <Link href={`/control-plane/jobs/${encodeURIComponent(jobID)}`} className="text-blue-700 underline hover:text-blue-900">
                  Open job
                </Link>
              ) : null}
              {runID ? (
                <Link href={`/control-plane/jobs/runs/${encodeURIComponent(runID)}`} className="text-blue-700 underline hover:text-blue-900">
                  Open run
                </Link>
              ) : null}
            </div>
          </li>
        );
      }}
    />
  );
}

export function FailedSyncsQueueClient() {
  const { rows, err, busy, refresh } = useFailedRuns("ingestion_runs");
  return (
    <QueueShell<Json>
      load={refresh}
      rows={rows}
      err={err}
      busy={busy}
      emptyMessage="No recent failed source syncs."
      countLabel={(n) => `${n} recent failed sync(s)`}
      renderRow={(r, i) => {
        const runID = asStr(r.id);
        const feedID = asStr(r.source_feed_id);
        const startedAt = asStr(r.started_at);
        return (
          <li key={runID || i} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
            <div>
              <StatusBadge status={asStr(r.status) || "failed"} />
              <span className="ml-2 text-gray-700">
                Feed <span className="font-mono text-xs">{shortId(feedID)}</span>
              </span>
              <span className="ml-2 text-xs text-gray-500">started {fmtTime(startedAt)}</span>
            </div>
            {feedID ? (
              <Link href={`/control-plane/sources/feeds/${encodeURIComponent(feedID)}/sync`} className="text-sm text-blue-700 underline hover:text-blue-900">
                Open sync history
              </Link>
            ) : null}
          </li>
        );
      }}
    />
  );
}

// ----- Policy exceptions -----------------------------------------------

export function PolicyExceptionsQueueClient() {
  const load = useCallback(() => apiJson<Json[]>("/governance/policy-exceptions"), []);
  const { rows, err, busy, refresh } = useQueue(load);
  return (
    <QueueShell<Json>
      load={refresh}
      rows={rows}
      err={err}
      busy={busy}
      emptyMessage="No active policy exceptions."
      countLabel={(n) => `${n} active exception(s)`}
      renderRow={(r, i) => {
        const id = asStr(r.id);
        const targetType = asStr(r.target_type);
        const targetID = asStr(r.target_id);
        const reason = asStr(r.reason || r.note);
        return (
          <li key={id || i} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm">
            <div>
              <StatusBadge status={asStr(r.status) || "active"} />
              <span className="ml-2 text-gray-700">
                {targetType || "—"} · <span className="font-mono text-xs">{shortId(targetID)}</span>
              </span>
              {reason ? <p className="mt-1 text-xs text-gray-600">{reason}</p> : null}
            </div>
            {targetType === "entity" && targetID ? (
              <Link href={`/entities/${encodeURIComponent(targetID)}`} className="text-sm text-blue-700 underline hover:text-blue-900">
                Open entity
              </Link>
            ) : null}
          </li>
        );
      }}
    />
  );
}
