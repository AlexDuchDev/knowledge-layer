"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { apiJson } from "@/lib/api";

type Task = {
  id: string;
  domain_id: string;
  title: string;
  description: string;
  review_status: string;
  priority: string;
  assignee_email?: string | null;
  assignee_display?: string | null;
  deadline_date?: string | null;
};

export default function MeetingTaskDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = typeof params.id === "string" ? params.id : "";
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [task, setTask] = useState<Task | null>(null);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState("medium");
  const [deadline, setDeadline] = useState("");

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

  const load = useCallback(() => {
    if (!id) return;
    void run(async () => {
      const t = await apiJson<Task>(`/extracted-meeting-tasks/${encodeURIComponent(id)}`);
      setTask(t);
      setTitle(t.title);
      setDescription(t.description ?? "");
      setPriority(t.priority || "medium");
      if (t.deadline_date) {
        const d = new Date(t.deadline_date);
        if (!Number.isNaN(d.getTime())) setDeadline(d.toISOString().slice(0, 10));
        else setDeadline("");
      } else setDeadline("");
    });
  }, [run, id]);

  useEffect(() => {
    load();
  }, [load]);

  const isDraft = task?.review_status === "draft";

  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <AppBreadcrumb
        items={[
          { label: "Home", href: "/" },
          { label: "Meeting tasks", href: "/meeting-tasks" },
          { label: id.slice(0, 8) + "…" },
        ]}
      />
      <h1 className="mt-4 text-2xl font-semibold tracking-tight">Extracted meeting task</h1>
      {err ? <div className="my-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      {!task ? (
        <p className="mt-4 text-sm text-neutral-600">Loading…</p>
      ) : (
        <div className="mt-6 space-y-4">
          <p className="text-sm text-neutral-600">
            Status: <span className="font-medium text-neutral-900">{task.review_status}</span>
          </p>
          <label className="block text-sm">
            <span className="text-xs font-medium text-neutral-600">Title</span>
            <input
              className="mt-1 w-full rounded border px-3 py-2 text-sm"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              disabled={busy || !isDraft}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs font-medium text-neutral-600">Description</span>
            <textarea
              className="mt-1 w-full rounded border px-3 py-2 text-sm"
              rows={5}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              disabled={busy || !isDraft}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs font-medium text-neutral-600">Priority</span>
            <select className="mt-1 rounded border px-2 py-1 text-sm" value={priority} onChange={(e) => setPriority(e.target.value)} disabled={busy || !isDraft}>
              <option value="low">low</option>
              <option value="medium">medium</option>
              <option value="high">high</option>
            </select>
          </label>
          <label className="block text-sm">
            <span className="text-xs font-medium text-neutral-600">Deadline (YYYY-MM-DD)</span>
            <input
              type="date"
              className="mt-1 rounded border px-3 py-2 text-sm"
              value={deadline}
              onChange={(e) => setDeadline(e.target.value)}
              disabled={busy || !isDraft}
            />
          </label>
          {isDraft ? (
            <div className="flex flex-wrap gap-2 pt-2">
              <button
                type="button"
                className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    const body: Record<string, unknown> = { title, description, priority };
                    if (deadline.trim()) body.deadline_date = deadline.trim();
                    await apiJson(`/extracted-meeting-tasks/${encodeURIComponent(id)}`, { method: "PATCH", body: JSON.stringify(body) });
                    load();
                  })
                }
              >
                Save draft
              </button>
              <button
                type="button"
                className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    await apiJson(`/extracted-meeting-tasks/${encodeURIComponent(id)}/confirm-no-edit`, { method: "POST" });
                    load();
                  })
                }
              >
                Confirm (no edits)
              </button>
              <button
                type="button"
                className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    const body: Record<string, unknown> = { title, description, priority };
                    if (deadline.trim()) body.deadline_date = deadline.trim();
                    await apiJson(`/extracted-meeting-tasks/${encodeURIComponent(id)}`, { method: "PATCH", body: JSON.stringify(body) });
                    await apiJson(`/extracted-meeting-tasks/${encodeURIComponent(id)}/confirm-after-edit`, { method: "POST" });
                    load();
                  })
                }
              >
                Save &amp; confirm
              </button>
              <button
                type="button"
                className="rounded-md border border-red-200 bg-red-50 px-3 py-1.5 text-sm text-red-900 disabled:opacity-50"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    await apiJson(`/extracted-meeting-tasks/${encodeURIComponent(id)}/reject`, { method: "POST" });
                    load();
                  })
                }
              >
                Reject
              </button>
            </div>
          ) : null}
          <p className="pt-4 text-sm">
            <Link href="/meeting-tasks" className="text-blue-700 underline">
              Back to list
            </Link>
            {" · "}
            <button type="button" className="text-blue-700 underline" onClick={() => router.push(`/entities?domain_id=${encodeURIComponent(task.domain_id)}`)}>
              Entities in domain
            </button>
          </p>
        </div>
      )}
    </main>
  );
}
