"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { WorkflowNextSteps } from "@/components/WorkflowNextSteps";
import { apiJson } from "@/lib/api";

type ReviewTask = { id: string; target_id: string; status: string };
type FailedRuns = { ingestion_runs: { id: string; error_message?: string }[]; job_runs: { id: string; error_message?: string }[] };
type FollowRow = { id: string; scope_type: string; ref_id: string; entity_type?: string };

export default function NotificationsPage() {
  const [reviews, setReviews] = useState<ReviewTask[] | null>(null);
  const [failed, setFailed] = useState<FailedRuns | null>(null);
  const [follows, setFollows] = useState<FollowRow[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const [r, f, fol] = await Promise.all([
          apiJson<ReviewTask[]>("/review-tasks").catch(() => []),
          apiJson<FailedRuns>("/ops/failed-runs").catch(() => ({ ingestion_runs: [], job_runs: [] })),
          apiJson<FollowRow[]>("/me/follows").catch(() => []),
        ]);
        const open = r.filter((x) => x.status === "pending" || x.status === "in_progress").slice(0, 20);
        setReviews(open);
        setFailed(f);
        setFollows(fol);
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
      }
    })();
  }, []);

  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Notifications" }]} />
      <div className="mt-3 rounded-md border border-neutral-200 bg-white px-3 py-2">
        <WorkflowNextSteps />
      </div>
      <h1 className="text-2xl font-semibold">Workflow notifications</h1>
      <p className="mt-2 text-sm text-neutral-600">Open reviews and recent failed runs (permission-gated endpoints).</p>
      {err ? <p className="mt-4 text-sm text-amber-800">{err}</p> : null}

      <section className="mt-8">
        <h2 className="text-sm font-semibold text-neutral-900">Followed scopes</h2>
        <p className="mt-1 text-xs text-neutral-600">
          Surfacing preferences — Home and this list reflect what you follow. This does not grant permissions.
        </p>
        <ul className="mt-2 space-y-2 text-sm">
          {(follows ?? []).length === 0 ? <li className="text-neutral-600">None yet. Follow a topic hub or a knowledge slice (with domain selected).</li> : null}
          {(follows ?? []).map((f) => (
            <li key={f.id} className="rounded border border-neutral-200 p-2 text-xs">
              <span className="font-medium text-neutral-800">{f.scope_type}</span>
              {f.scope_type === "content_hub" ? (
                <>
                  {" "}
                  ·{" "}
                  <Link href={`/hubs/${f.ref_id}`} className="text-blue-700 underline">
                    Open hub
                  </Link>
                </>
              ) : null}
              {f.scope_type === "knowledge_topic" ? (
                <>
                  {" "}
                  · type <code className="rounded bg-neutral-100 px-1">{f.entity_type ?? "—"}</code> in domain{" "}
                  <code className="rounded bg-neutral-100 px-1">{f.ref_id.slice(0, 8)}…</code>
                </>
              ) : null}
              {f.scope_type === "domain" ? (
                <>
                  {" "}
                  · domain <code className="rounded bg-neutral-100 px-1">{f.ref_id.slice(0, 8)}…</code>
                </>
              ) : null}
              {f.scope_type === "digest_stream" ? (
                <>
                  {" "}
                  · digest stream in domain <code className="rounded bg-neutral-100 px-1">{f.ref_id.slice(0, 8)}…</code>
                </>
              ) : null}
            </li>
          ))}
        </ul>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-semibold text-neutral-900">Open review tasks</h2>
        <ul className="mt-2 space-y-2 text-sm">
          {(reviews ?? []).length === 0 ? <li className="text-neutral-600">None loaded or none open.</li> : null}
          {(reviews ?? []).map((t) => (
            <li key={t.id} className="rounded border border-neutral-200 p-2">
              <Link href="/governance" className="text-blue-700 underline">
                Review {t.id.slice(0, 8)}…
              </Link>
              <span className="ml-2 text-xs text-neutral-500">{t.status}</span>
            </li>
          ))}
        </ul>
      </section>

      <section className="mt-8">
        <h2 className="text-sm font-semibold text-neutral-900">Failed runs (ops)</h2>
        <p className="mt-1 text-xs text-neutral-600">Requires identity-admin style access; may be empty.</p>
        <ul className="mt-2 space-y-1 text-xs text-neutral-700">
          {(failed?.ingestion_runs ?? []).map((x) => (
            <li key={x.id}>Ingestion {x.id}: {x.error_message ?? "error"}</li>
          ))}
          {(failed?.job_runs ?? []).map((x) => (
            <li key={x.id}>Job {x.id}: {x.error_message ?? "error"}</li>
          ))}
        </ul>
      </section>
    </main>
  );
}
