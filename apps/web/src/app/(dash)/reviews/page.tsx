"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { WorkflowNextSteps } from "@/components/WorkflowNextSteps";
import { apiJson } from "@/lib/api";

type Task = {
  id: string;
  target_type: string;
  target_id: string;
  status: string;
  created_at: string;
};

export default function ReviewsPage() {
  const [rows, setRows] = useState<Task[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setRows(await apiJson<Task[]>("/review-tasks"));
        setErr(null);
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setRows(null);
      }
    })();
  }, []);

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Reviews" }]} />
      <div className="mt-3 rounded-md border border-neutral-200 bg-white px-3 py-2">
        <WorkflowNextSteps />
      </div>
      <h1 className="text-2xl font-semibold tracking-tight">Reviews</h1>
      <p className="mt-1 text-sm text-neutral-600">Open review tasks. Actions (approve, request changes) run from governance tooling or API.</p>
      {err ? <div className="my-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      <ul className="mt-6 divide-y divide-neutral-200 rounded-lg border border-neutral-200 bg-white">
        {(rows ?? []).map((t) => (
          <li key={t.id} className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 text-sm">
            <div>
              <span className="font-medium text-neutral-900">{t.status}</span>
              <span className="ml-2 text-neutral-600">
                {t.target_type} · {t.target_id.slice(0, 8)}…
              </span>
            </div>
            {t.target_type === "entity" ? (
              <Link href={`/entities/${t.target_id}`} className="text-blue-700 underline">
                Open entity
              </Link>
            ) : null}
          </li>
        ))}
      </ul>
      {rows && rows.length === 0 ? <p className="mt-4 text-sm text-neutral-600">No review tasks.</p> : null}
      <p className="mt-8 text-sm">
        <Link href="/governance" className="text-blue-700 underline">
          Governance Center
        </Link>
      </p>
    </main>
  );
}
