"use client";

import { useCallback, useState } from "react";
import Link from "next/link";
import { QueuePageTemplate } from "@/components/templates/QueuePageTemplate";
import { apiBase, apiJson, principalUserId } from "@/lib/api";

type Json = Record<string, unknown>;

const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function isUuid(s: string): boolean {
  return UUID_RE.test(s.trim());
}

export default function GovernancePage() {
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const run = useCallback(async (key: string, fn: () => Promise<void>) => {
    setErr(null);
    setBusy(key);
    try {
      await fn();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }, []);

  const description = (
    <span className="text-sm text-neutral-600">
      Pilot ops: queues, policy exceptions, owners, feedback. API <code className="rounded bg-neutral-100 px-1">{apiBase()}</code> · principal{" "}
      <code className="rounded bg-neutral-100 px-1">{principalUserId()}</code>
    </span>
  );

  return (
    <QueuePageTemplate
      title="Governance Center"
      description={description}
      actions={
        <div className="flex flex-wrap gap-3 text-sm">
          <Link href="/" className="text-blue-700 underline">
            Home
          </Link>
          <Link href="/control-plane/governance" className="text-blue-700 underline">
            Control plane
          </Link>
        </div>
      }
    >
      <nav className="mb-8 flex flex-wrap gap-2 text-sm">
        <Link href="/governance/editorial" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900">
          Publishing queue
        </Link>
        <Link href="/governance/workflow-metrics" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900">
          Workflow metrics
        </Link>
        <Link href="/governance/freshness-risk" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900">
          Freshness risk
        </Link>
        <Link href="/ops/answer-diagnostics" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900">
          Answer diagnostics
        </Link>
        <Link href="/ops/search-insights" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900">
          Search insights
        </Link>
        <Link href="/notifications" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900">
          Notifications
        </Link>
      </nav>

      {err ? (
        <div className="mb-6 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">
          {err}
        </div>
      ) : null}

      <div className="flex flex-col gap-12">
        <SourceFeedPreviewSection run={run} busy={busy} />
        <OverdueSection run={run} busy={busy} />
        <ApprovalQueueSection run={run} busy={busy} />
        <PolicyExceptionsSection run={run} busy={busy} />
        <MissingOwnersSection run={run} busy={busy} />
        <AnswerFeedbackSection run={run} busy={busy} />
        <KnowledgeJobsSection run={run} busy={busy} />
        <UpkeepSuggestionsSection run={run} busy={busy} />
        <StaleContentSection run={run} busy={busy} />
        <AISummarizeSection run={run} busy={busy} />
        <OpsHealthSection run={run} busy={busy} />
        <SearchExpandHint />
      </div>
    </QueuePageTemplate>
  );
}

function SourceFeedPreviewSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [feeds, setFeeds] = useState<Json[] | null>(null);
  const [feedId, setFeedId] = useState("");
  const [preview, setPreview] = useState<Json | null>(null);

  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Source feed preview</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Load feeds (draft feeds can be previewed without activation).
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("feeds", async () => {
              const list = await apiJson<Json[]>("/source-feeds");
              setFeeds(list);
            })
          }
        >
          {busy === "feeds" ? "Loading…" : "List source feeds"}
        </button>
      </div>
      {feeds ? (
        <ul className="mt-3 max-h-40 overflow-auto text-sm">
          {feeds.map((f) => (
            <li key={String(f.id)} className="border-b border-neutral-100 py-1">
              <button
                type="button"
                className="text-left text-blue-700 underline"
                onClick={() => setFeedId(String(f.id))}
              >
                {String(f.display_name ?? f.id)} — {String(f.status ?? "")}
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      <div className="mt-4 flex max-w-xl flex-col gap-2">
        <label className="text-sm font-medium text-neutral-800">Feed ID</label>
        <input
          className="rounded border border-neutral-300 px-2 py-1.5 text-sm"
          value={feedId}
          onChange={(e) => setFeedId(e.target.value)}
          placeholder="uuid"
        />
        <button
          type="button"
          className="w-fit rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null || !feedId}
          onClick={() =>
            run("preview", async () => {
              const body = await apiJson<Json>(`/source-feeds/${feedId}/preview`, {
                method: "POST",
                body: "{}",
              });
              setPreview(body);
            })
          }
        >
          {busy === "preview" ? "Preview…" : "Preview (no activation)"}
        </button>
      </div>
      {preview ? (
        <pre className="mt-3 max-h-64 overflow-auto rounded bg-neutral-50 p-3 text-xs">
          {JSON.stringify(preview, null, 2)}
        </pre>
      ) : null}
    </section>
  );
}

function OverdueSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [rows, setRows] = useState<Json[] | null>(null);
  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Overdue reviews</h2>
      <p className="mt-1 text-sm text-neutral-600">Tasks past due_at (requires publish-capable principal).</p>
      <button
        type="button"
        className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy !== null}
        onClick={() =>
          run("overdue", async () => {
            const list = await apiJson<Json[]>("/governance/reviews/overdue");
            setRows(list);
          })
        }
      >
        {busy === "overdue" ? "Loading…" : "Load overdue queue"}
      </button>
      {rows ? (
        <TaskTable rows={rows} />
      ) : null}
    </section>
  );
}

function ApprovalQueueSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [rows, setRows] = useState<Json[] | null>(null);
  const [taskId, setTaskId] = useState("");
  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Approval queue</h2>
      <p className="mt-1 text-sm text-neutral-600">Entity review tasks awaiting decision.</p>
      <button
        type="button"
        className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy !== null}
        onClick={() =>
          run("approval", async () => {
            const list = await apiJson<Json[]>("/governance/approval-queue");
            setRows(list);
          })
        }
      >
        {busy === "approval" ? "Loading…" : "Load approval queue"}
      </button>
      {rows ? (
        <TaskTable rows={rows} />
      ) : null}
      <div className="mt-6 rounded-lg border border-neutral-200 p-3">
        <h3 className="text-sm font-medium">Review actions</h3>
        <p className="mt-1 text-xs text-neutral-600">Paste a review_task id from the queue or GET /review-tasks.</p>
        <input
          className="mt-2 w-full max-w-xl rounded border px-2 py-1 font-mono text-xs"
          placeholder="review_task id"
          value={taskId}
          onChange={(e) => setTaskId(e.target.value)}
        />
        <div className="mt-2 flex flex-wrap gap-2">
          <button
            type="button"
            className="rounded border border-neutral-300 px-2 py-1 text-xs disabled:opacity-50"
            disabled={busy !== null || !isUuid(taskId)}
            onClick={() =>
              run("rv-start", async () => {
                await apiJson(`/review-tasks/${taskId.trim()}/start`, { method: "POST" });
              })
            }
          >
            start
          </button>
          <button
            type="button"
            className="rounded bg-green-800 px-2 py-1 text-xs text-white disabled:opacity-50"
            disabled={busy !== null || !isUuid(taskId)}
            onClick={() =>
              run("rv-ok", async () => {
                await apiJson(`/review-tasks/${taskId.trim()}/approve`, { method: "POST", body: JSON.stringify({}) });
              })
            }
          >
            approve
          </button>
          <button
            type="button"
            className="rounded bg-amber-800 px-2 py-1 text-xs text-white disabled:opacity-50"
            disabled={busy !== null || !isUuid(taskId)}
            onClick={() =>
              run("rv-ch", async () => {
                await apiJson(`/review-tasks/${taskId.trim()}/request-changes`, {
                  method: "POST",
                  body: JSON.stringify({ note: "please revise" }),
                });
              })
            }
          >
            request changes
          </button>
          <button
            type="button"
            className="rounded bg-red-800 px-2 py-1 text-xs text-white disabled:opacity-50"
            disabled={busy !== null || !isUuid(taskId)}
            onClick={() =>
              run("rv-no", async () => {
                await apiJson(`/review-tasks/${taskId.trim()}/reject`, { method: "POST", body: JSON.stringify({}) });
              })
            }
          >
            reject
          </button>
        </div>
      </div>
    </section>
  );
}

function TaskTable({ rows }: { rows: Json[] }) {
  if (rows.length === 0) {
    return <p className="mt-3 text-sm text-neutral-500">No rows.</p>;
  }
  const keys = Object.keys(rows[0] ?? {});
  return (
    <div className="mt-3 overflow-x-auto rounded border border-neutral-200">
      <table className="min-w-full text-left text-xs">
        <thead className="bg-neutral-50">
          <tr>
            {keys.map((k) => (
              <th key={k} className="whitespace-nowrap px-2 py-1 font-medium text-neutral-700">
                {k}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-t border-neutral-100">
              {keys.map((k) => (
                <td key={k} className="whitespace-nowrap px-2 py-1 text-neutral-800">
                  <JsonCell value={r[k]} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function JsonCell({ value }: { value: unknown }) {
  if (value === null || value === undefined) return <span className="text-neutral-500">—</span>;
  if (typeof value !== "object") return <span>{String(value)}</span>;
  const text = JSON.stringify(value, null, 2);
  const preview = text.length > 80 ? `${text.slice(0, 80)}…` : text;
  return (
    <details>
      <summary className="cursor-pointer select-none text-neutral-700">{preview}</summary>
      <pre className="mt-2 max-h-56 overflow-auto rounded bg-neutral-50 p-2 text-[11px] text-neutral-900">
        {text}
      </pre>
    </details>
  );
}

function PolicyExceptionsSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [rows, setRows] = useState<Json[] | null>(null);
  const [targetType, setTargetType] = useState("entity");
  const [targetId, setTargetId] = useState("");
  const [overrideType, setOverrideType] = useState("allow");
  const [reason, setReason] = useState("");
  const [selectedId, setSelectedId] = useState("");

  const reload = () =>
    run("policies", async () => {
      const list = await apiJson<Json[]>("/governance/policy-exceptions");
      setRows(list);
    });

  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Policy exceptions</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Exceptions are sensitive. Create requires reason. Review activates a pending exception. Revoke disables it.
      </p>
      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null}
          onClick={() => reload()}
        >
          {busy === "policies" ? "Loading…" : "Refresh list"}
        </button>
      </div>
      {rows ? (
        <div className="mt-3 overflow-x-auto rounded border border-neutral-200">
          <table className="min-w-full text-left text-xs">
            <thead className="bg-neutral-50">
              <tr>
                {["id", "target_type", "target_id", "override_type", "effective_status", "actions"].map((k) => (
                  <th key={k} className="whitespace-nowrap px-2 py-1 font-medium text-neutral-700">
                    {k}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((r, i) => {
                const id = typeof r.id === "string" ? r.id : "";
                return (
                  <tr key={i} className="border-t border-neutral-100">
                    <td className="whitespace-nowrap px-2 py-1">
                      <button
                        type="button"
                        className="text-blue-700 underline"
                        onClick={() => setSelectedId(id)}
                      >
                        {id || "—"}
                      </button>
                    </td>
                    <td className="whitespace-nowrap px-2 py-1">{String(r.target_type ?? "—")}</td>
                    <td className="whitespace-nowrap px-2 py-1">{String(r.target_id ?? "—")}</td>
                    <td className="whitespace-nowrap px-2 py-1">{String(r.override_type ?? "—")}</td>
                    <td className="whitespace-nowrap px-2 py-1">{String(r.effective_status ?? r.status ?? "—")}</td>
                    <td className="whitespace-nowrap px-2 py-1">
                      <div className="flex gap-2">
                        <button
                          type="button"
                          className="rounded bg-neutral-900 px-2 py-1 text-white disabled:opacity-50"
                          disabled={busy !== null || !id}
                          onClick={() =>
                            run(`policy-review-${id}`, async () => {
                              await apiJson(`/governance/policy-exceptions/${id}/review`, { method: "POST", body: "{}" });
                              await reload();
                            })
                          }
                        >
                          Review
                        </button>
                        <button
                          type="button"
                          className="rounded bg-red-700 px-2 py-1 text-white disabled:opacity-50"
                          disabled={busy !== null || !id}
                          onClick={() =>
                            run(`policy-revoke-${id}`, async () => {
                              await apiJson(`/governance/policy-exceptions/${id}/revoke`, { method: "POST", body: "{}" });
                              await reload();
                            })
                          }
                        >
                          Revoke
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}
      <div className="mt-6 grid max-w-xl gap-2 text-sm">
        <input
          className="rounded border px-2 py-1"
          placeholder="target_type (e.g. entity)"
          value={targetType}
          onChange={(e) => setTargetType(e.target.value)}
        />
        <input
          className="rounded border px-2 py-1"
          placeholder="target_id uuid"
          value={targetId}
          onChange={(e) => setTargetId(e.target.value)}
        />
        <input
          className="rounded border px-2 py-1"
          placeholder="override_type (allow | deny)"
          value={overrideType}
          onChange={(e) => setOverrideType(e.target.value)}
        />
        <textarea
          className="rounded border px-2 py-1"
          placeholder="reason (required)"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={2}
        />
        <button
          type="button"
          className="w-fit rounded-md bg-blue-800 px-3 py-1.5 text-white disabled:opacity-50"
          disabled={busy !== null || !isUuid(targetId) || reason.trim() === ""}
          onClick={() =>
            run("policy-create", async () => {
              await apiJson("/governance/policy-exceptions", {
                method: "POST",
                body: JSON.stringify({
                  target_type: targetType,
                  target_id: targetId,
                  override_type: overrideType,
                  policy_payload: {},
                  reason,
                }),
              });
              await reload();
            })
          }
        >
          Create exception
        </button>
        {!isUuid(targetId) && targetId !== "" ? (
          <p className="text-xs text-neutral-600">target_id must be a valid UUID.</p>
        ) : null}
        {selectedId ? (
          <p className="text-xs text-neutral-600">
            Selected exception: <code className="rounded bg-neutral-100 px-1">{selectedId}</code>
          </p>
        ) : null}
        <p className="text-xs text-neutral-600">
          Revoke is a side-effecting action. Prefer review before activate; keep exceptions rare and reasoned.
        </p>
      </div>
    </section>
  );
}

function MissingOwnersSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [rows, setRows] = useState<Json[] | null>(null);
  const [resourceType, setResourceType] = useState("entity");
  const [resourceId, setResourceId] = useState("");
  const [ownerId, setOwnerId] = useState(principalUserId());

  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Missing owners</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Assigning an owner updates the underlying object (side effect). Ensure the new owner is correct for the domain.
      </p>
      <button
        type="button"
        className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy !== null}
        onClick={() =>
          run("owners", async () => {
            const list = await apiJson<Json[]>("/governance/missing-owners");
            setRows(list);
          })
        }
      >
        {busy === "owners" ? "Loading…" : "List missing owners"}
      </button>
      {rows ? (
        <TaskTable rows={rows} />
      ) : null}
      <div className="mt-6 grid max-w-xl gap-2 text-sm">
        <input
          className="rounded border px-2 py-1"
          placeholder="resource_type (entity | source_feed | knowledge_job)"
          value={resourceType}
          onChange={(e) => setResourceType(e.target.value)}
        />
        <input
          className="rounded border px-2 py-1"
          placeholder="resource_id"
          value={resourceId}
          onChange={(e) => setResourceId(e.target.value)}
        />
        <input
          className="rounded border px-2 py-1"
          placeholder="new owner_id"
          value={ownerId}
          onChange={(e) => setOwnerId(e.target.value)}
        />
        <button
          type="button"
          className="w-fit rounded-md bg-blue-800 px-3 py-1.5 text-white disabled:opacity-50"
          disabled={busy !== null || !isUuid(resourceId) || !isUuid(ownerId)}
          onClick={() =>
            run("assign", async () => {
              await apiJson("/governance/missing-owners/assign", {
                method: "POST",
                body: JSON.stringify({
                  resource_type: resourceType,
                  resource_id: resourceId,
                  owner_id: ownerId,
                }),
              });
            })
          }
        >
          Assign owner
        </button>
        {!isUuid(resourceId) && resourceId !== "" ? (
          <p className="text-xs text-neutral-600">resource_id must be a valid UUID.</p>
        ) : null}
        {!isUuid(ownerId) && ownerId !== "" ? (
          <p className="text-xs text-neutral-600">owner_id must be a valid UUID.</p>
        ) : null}
      </div>
    </section>
  );
}

function AnswerFeedbackSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [rows, setRows] = useState<Json[] | null>(null);
  const [traceId, setTraceId] = useState("pilot-trace-1");
  const [kind, setKind] = useState("useful");

  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Answer feedback</h2>
      <p className="mt-1 text-sm text-neutral-600">Submit feedback; admin list requires publish gate.</p>
      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("fb-list", async () => {
              const list = await apiJson<Json[]>("/answer-feedback");
              setRows(list);
            })
          }
        >
          {busy === "fb-list" ? "Loading…" : "Load feedback (admin)"}
        </button>
      </div>
      {rows ? (
        <TaskTable rows={rows} />
      ) : null}
      <div className="mt-6 grid max-w-xl gap-2 text-sm">
        <input className="rounded border px-2 py-1" value={traceId} onChange={(e) => setTraceId(e.target.value)} />
        <select className="rounded border px-2 py-1" value={kind} onChange={(e) => setKind(e.target.value)}>
          {["useful", "not_useful", "stale", "weak_citations", "incorrect", "incomplete"].map((k) => (
            <option key={k} value={k}>
              {k}
            </option>
          ))}
        </select>
        <button
          type="button"
          className="w-fit rounded-md bg-blue-800 px-3 py-1.5 text-white disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("fb-post", async () => {
              await apiJson("/answer-feedback", {
                method: "POST",
                body: JSON.stringify({ trace_id: traceId, feedback_kind: kind }),
              });
            })
          }
        >
          Submit feedback
        </button>
      </div>
    </section>
  );
}

type JobTemplateRow = {
  id: string;
  title: string;
  description: string;
  job_type: string;
  default_name: string;
  default_review_required: boolean;
  default_output_sensitivity: number;
  source_scope_hint_json: string;
  default_config_preview_json: string;
};

function KnowledgeJobsSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [templates, setTemplates] = useState<JobTemplateRow[] | null>(null);
  const [selectedId, setSelectedId] = useState<string>("");
  const [nameOverride, setNameOverride] = useState("");
  const [scopeJson, setScopeJson] = useState('{\n  "domain_id": "",\n  "source_feed_id": ""\n}');
  const [created, setCreated] = useState<Json | null>(null);
  const [jobs, setJobs] = useState<Json[] | null>(null);

  const selected = templates?.find((t) => t.id === selectedId) ?? null;

  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Knowledge jobs</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Browse templates (v1 library), then create a job with optional name override and JSON scope. Weekly digest
        needs <code className="rounded bg-neutral-100 px-1">domain_id</code> and{" "}
        <code className="rounded bg-neutral-100 px-1">source_feed_id</code> in scope.
      </p>
      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("job-templates", async () => {
              const list = await apiJson<JobTemplateRow[]>("/knowledge-job-templates");
              setTemplates(list);
              if (!selectedId && list[0]) {
                setSelectedId(list[0].id);
              }
            })
          }
        >
          {busy === "job-templates" ? "Loading…" : "Load job templates"}
        </button>
        <button
          type="button"
          className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("job-list", async () => {
              const list = await apiJson<Json[]>("/knowledge-jobs");
              setJobs(list);
            })
          }
        >
          {busy === "job-list" ? "Loading…" : "List knowledge jobs"}
        </button>
      </div>

      {templates && templates.length > 0 ? (
        <div className="mt-6 max-w-2xl space-y-3 text-sm">
          <label className="block font-medium text-neutral-800">Template</label>
          <select
            className="w-full rounded border border-neutral-300 px-2 py-2"
            value={selectedId}
            onChange={(e) => setSelectedId(e.target.value)}
          >
            {templates.map((t) => (
              <option key={t.id} value={t.id}>
                {t.title} ({t.job_type})
              </option>
            ))}
          </select>
          {selected ? (
            <p className="text-neutral-600">
              {selected.description} Default config preview:{" "}
              <code className="rounded bg-neutral-50 px-1 text-xs">{selected.default_config_preview_json}</code>
            </p>
          ) : null}
          <label className="block font-medium text-neutral-800">Name override (optional)</label>
          <input
            className="w-full rounded border border-neutral-300 px-2 py-1.5"
            value={nameOverride}
            onChange={(e) => setNameOverride(e.target.value)}
            placeholder="Leave blank to use template default"
          />
          <label className="block font-medium text-neutral-800">source_scope_json</label>
          <textarea
            className="w-full rounded border border-neutral-300 px-2 py-2 font-mono text-xs"
            rows={5}
            value={scopeJson}
            onChange={(e) => setScopeJson(e.target.value)}
          />
          <button
            type="button"
            className="rounded-md bg-blue-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy !== null || !selectedId}
            onClick={() =>
              run("job-create", async () => {
                let scope: Json = {};
                try {
                  scope = JSON.parse(scopeJson) as Json;
                } catch {
                  throw new Error("source_scope_json must be valid JSON");
                }
                const body: Json = {
                  template_id: selectedId,
                  owner_id: principalUserId(),
                  source_scope_json: scope,
                };
                if (nameOverride.trim()) {
                  body.name = nameOverride.trim();
                }
                const job = await apiJson<Json>("/knowledge-jobs", {
                  method: "POST",
                  body: JSON.stringify(body),
                });
                setCreated(job);
              })
            }
          >
            {busy === "job-create" ? "Creating…" : "Create job from template"}
          </button>
        </div>
      ) : null}

      {created ? (
        <pre className="mt-4 max-h-64 overflow-auto rounded bg-neutral-50 p-3 text-xs">
          {JSON.stringify(created, null, 2)}
        </pre>
      ) : null}

      {jobs ? (
        <ul className="mt-4 max-h-48 overflow-auto text-sm">
          {jobs.map((j) => (
            <li key={String(j.id)} className="border-b border-neutral-100 py-1">
              <span className="font-medium">{String(j.name ?? "")}</span> — {String(j.job_type ?? "")} (
              {String(j.status ?? "")})
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function UpkeepSuggestionsSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [rows, setRows] = useState<Json[] | null>(null);
  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Content upkeep suggestions</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Heuristic hints (stale freshness, weak summary, missing links on decisions/policies). Review-only — no auto-edits. GET
        /governance/upkeep-suggestions
      </p>
      <button
        type="button"
        className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy !== null}
        onClick={() =>
          run("upkeep", async () => {
            setRows(await apiJson<Json[]>("/governance/upkeep-suggestions"));
          })
        }
      >
        {busy === "upkeep" ? "Loading…" : "Load upkeep suggestions"}
      </button>
      {rows ? (
        <ul className="mt-3 max-h-64 overflow-auto text-xs">
          {rows.map((r) => (
            <li key={String(r.entity_id)} className="border-b border-neutral-100 py-2">
              <Link href={`/entities/${String(r.entity_id)}`} className="font-medium text-blue-800 underline">
                {String(r.title)}
              </Link>
              <div className="text-neutral-600">
                reason={String(r.reason)} · {String(r.evidence)}
              </div>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function StaleContentSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [rows, setRows] = useState<Json[] | null>(null);
  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Stale content queue</h2>
      <p className="mt-1 text-sm text-neutral-600">GET /governance/stale-content (publish-scoped domains).</p>
      <button
        type="button"
        className="mt-3 rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy !== null}
        onClick={() =>
          run("stale", async () => {
            setRows(await apiJson<Json[]>("/governance/stale-content"));
          })
        }
      >
        {busy === "stale" ? "Loading…" : "Load stale entities"}
      </button>
      {rows ? (
        <ul className="mt-3 max-h-56 overflow-auto text-xs">
          {rows.map((r) => (
            <li key={String(r.id)} className="border-b border-neutral-100 py-1 font-mono">
              {String(r.title)} · freshness={String(r.freshness_status)} · {String(r.id)}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function AISummarizeSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [text, setText] = useState("");
  const [out, setOut] = useState<string | null>(null);
  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">AI summarize (governed)</h2>
      <p className="mt-1 text-sm text-neutral-600">POST /ai/summarize — identity admin only; audited.</p>
      <textarea
        className="mt-3 w-full max-w-2xl rounded border border-neutral-300 px-3 py-2 text-sm"
        rows={5}
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder="Paste text to summarize…"
      />
      <button
        type="button"
        className="mt-2 rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy !== null || !text.trim()}
        onClick={() =>
          run("summ", async () => {
            const res = await apiJson<{ summary: string }>("/ai/summarize", {
              method: "POST",
              body: JSON.stringify({ text: text.trim() }),
            });
            setOut(res.summary);
          })
        }
      >
        {busy === "summ" ? "Running…" : "Summarize"}
      </button>
      {out ? <pre className="mt-3 max-h-48 overflow-auto rounded bg-neutral-50 p-3 text-sm whitespace-pre-wrap">{out}</pre> : null}
    </section>
  );
}

function OpsHealthSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [health, setHealth] = useState<Json | null>(null);
  const [failed, setFailed] = useState<Json | null>(null);
  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Ops</h2>
      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("ops-h", async () => {
              setHealth(await apiJson<Json>("/ops/health"));
            })
          }
        >
          GET /ops/health
        </button>
        <button
          type="button"
          className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("ops-f", async () => {
              setFailed(await apiJson<Json>("/ops/failed-runs"));
            })
          }
        >
          GET /ops/failed-runs
        </button>
      </div>
      {health ? (
        <pre className="mt-3 max-h-32 overflow-auto rounded bg-neutral-50 p-2 text-xs">{JSON.stringify(health, null, 2)}</pre>
      ) : null}
      {failed ? (
        <pre className="mt-3 max-h-40 overflow-auto rounded bg-neutral-50 p-2 text-xs">{JSON.stringify(failed, null, 2)}</pre>
      ) : null}
    </section>
  );
}

function SearchExpandHint() {
  return (
    <section className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Relation-aware search</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Use GET <code className="rounded bg-neutral-100 px-1">/search?expand_relations=1</code> with normal filters.
        Linked entities appear with <code className="rounded bg-neutral-100 px-1">relation_expansion</code> metadata when
        they stay in domains you can access.
      </p>
    </section>
  );
}
