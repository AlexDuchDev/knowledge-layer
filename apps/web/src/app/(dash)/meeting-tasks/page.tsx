"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { apiJson } from "@/lib/api";

type DomainRow = { id: string; name: string };

type ExtractedTask = {
  id: string;
  title: string;
  review_status: string;
  priority: string;
  created_at: string;
};

type Metrics = Record<string, number>;

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

export default function MeetingTasksPage() {
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [domains, setDomains] = useState<DomainRow[] | null>(null);
  const [domainId, setDomainId] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [rows, setRows] = useState<ExtractedTask[] | null>(null);
  const [metrics, setMetrics] = useState<Metrics | null>(null);

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

  useEffect(() => {
    void run(async () => {
      const d = await apiJson<DomainRow[]>("/domains");
      setDomains(d);
      if (d.length > 0) {
        setDomainId((cur) => cur || d[0].id);
      }
    });
  }, [run]);

  const loadList = useCallback(() => {
    if (!domainId.trim()) return;
    void run(async () => {
      const q = new URLSearchParams({ limit: "100" });
      if (statusFilter.trim()) q.set("review_status", statusFilter.trim());
      setRows(await apiJson<ExtractedTask[]>(`/domains/${encodeURIComponent(domainId)}/extracted-meeting-tasks?${q}`));
      const m = await apiJson<Metrics>(`/domains/${encodeURIComponent(domainId)}/second-brain-metrics`);
      setMetrics(m);
    });
  }, [run, domainId, statusFilter]);

  useEffect(() => {
    if (domainId) loadList();
  }, [domainId, loadList]);

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Meeting tasks (extracted)" }]} />
      <h1 className="text-2xl font-semibold tracking-tight">Meeting tasks (extracted)</h1>
      <p className="mt-1 text-sm text-neutral-600">
        Review and confirm tasks extracted from meetings (Second Brain). Data comes from <code className="text-xs">extracted_meeting_tasks</code>.
      </p>
      {err ? <div className="my-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}

      <section className="mt-6 rounded-lg border border-neutral-200 p-4">
        <div className="flex flex-wrap items-end gap-3">
          <label className="text-sm">
            <span className="block text-xs font-medium text-neutral-600">Domain</span>
            <select className="mt-1 rounded border px-2 py-1 text-sm" value={domainId} onChange={(e) => setDomainId(e.target.value)} disabled={busy}>
              <option value="">Select…</option>
              {(domains ?? []).map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm">
            <span className="block text-xs font-medium text-neutral-600">Status</span>
            <select
              className="mt-1 rounded border px-2 py-1 text-sm"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              disabled={busy}
            >
              <option value="">all</option>
              <option value="draft">draft</option>
              <option value="confirmed">confirmed</option>
              <option value="edited">edited</option>
              <option value="rejected">rejected</option>
            </select>
          </label>
          <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50" disabled={busy || !domainId} onClick={() => loadList()}>
            Refresh
          </button>
        </div>
        {metrics && domainId ? (
          <p className="mt-3 text-xs text-neutral-600">
            Counts — draft: {asStr(metrics.draft)}, confirmed: {asStr(metrics.confirmed)}, edited: {asStr(metrics.edited)}, rejected:{" "}
            {asStr(metrics.rejected)}
          </p>
        ) : null}
      </section>

      <ul className="mt-6 divide-y divide-neutral-200 rounded-lg border border-neutral-200 bg-white">
        {(rows ?? []).map((t) => (
          <li key={t.id} className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 text-sm">
            <div>
              <span className="font-medium text-neutral-900">{t.title}</span>
              <span className="ml-2 text-neutral-500">
                {t.review_status} · {t.priority}
              </span>
            </div>
            <Link href={`/meeting-tasks/${t.id}`} className="text-blue-700 underline">
              Open
            </Link>
          </li>
        ))}
      </ul>
      {rows && rows.length === 0 ? <p className="mt-4 text-sm text-neutral-600">No tasks in this view.</p> : null}
      <p className="mt-8 text-sm text-neutral-600">
        <Link href="/settings" className="text-blue-700 underline">
          Settings
        </Link>{" "}
        — link Telegram / Mattermost IDs via API <code className="text-xs">GET/PUT /me/chat-links</code> (see docs).
      </p>
    </main>
  );
}
