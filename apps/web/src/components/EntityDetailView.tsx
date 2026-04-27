"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { formatTruthMode, TrustBadge } from "@/components/TrustBadge";
import { TrustExplanationDrawer } from "@/components/TrustExplanationDrawer";

export type Entity = {
  id: string;
  type: string;
  title: string;
  summary?: string | null;
  body?: string | null;
  owner_id?: string | null;
  domain_id: string;
  sensitivity_level: number;
  truth_mode: string;
  lifecycle_state: string;
  freshness_status: string;
  canonical_status: string;
  approval_status: string;
  external_ref?: string | null;
  access_policy_id?: string | null;
  created_at: string;
  updated_at: string;
};

export type EntityPayload = {
  entity_id: string;
  entity_type: string;
  payload_json: unknown;
  schema_version: number;
  created_at: string;
  updated_at: string;
};

export type ProvenanceRecord = {
  id: string;
  target_type: string;
  target_id: string;
  origin_type: string;
  origin_ref?: string | null;
  source_feed_id?: string | null;
  job_run_id?: string | null;
  created_by_id?: string | null;
  created_at: string;
};

export type EntityDetailResponse = {
  entity: Entity;
  payload: EntityPayload | null;
  provenance: ProvenanceRecord[];
  snapshot_at: string;
  source: string | null;
  open_in_source_url: string | null;
  content_preview: { text: string; truncated: boolean } | null;
  freshness_status: string;
  truth_mode: string;
  external_ref: string | null;
  owner_id: string | null;
  domain_id: string;
  sensitivity_level: number;
  lifecycle_state: string;
  canonical_status: string;
  approval_status: string;
  updated_at: string;
  created_at: string;
  access_policy_id: string | null;
};

export type RelatedEntityItem = {
  entity: Pick<Entity, "id" | "type" | "title" | "truth_mode" | "lifecycle_state" | "freshness_status" | "updated_at"> & {
    domain_id: string;
  };
  reason: string;
};

export type EntityEvidenceItem = {
  record: ProvenanceRecord;
  raw_artifact_ids?: string[];
  normalized_record_ids?: string[];
};

export type EntityEvidenceResponse = {
  entity_id: string;
  can_view_raw: boolean;
  can_view_normalized: boolean;
  evidence: EntityEvidenceItem[];
};

export type ContentBlockRow = {
  id: string;
  domain_id: string;
  title: string;
  body: string;
  truth_mode: string;
  lifecycle_state: string;
  approval_status: string;
  created_at: string;
  updated_at: string;
};

function KeyValue({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <div className="flex flex-wrap gap-x-2 gap-y-1 text-sm">
      <div className="w-40 shrink-0 text-neutral-600">{k}</div>
      <div className="min-w-0 text-neutral-900">{v}</div>
    </div>
  );
}

function renderEvidenceItems(items: EntityEvidenceItem[]) {
  return items.map((ev) => (
    <li key={ev.record.id} className="rounded border border-neutral-200 p-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div className="font-medium text-neutral-900">{ev.record.origin_type}</div>
        <div className="text-xs text-neutral-600">{new Date(ev.record.created_at).toISOString()}</div>
      </div>
      <div className="mt-2 grid gap-1 text-xs text-neutral-700">
        <div>
          origin_ref: <span className="break-all">{ev.record.origin_ref ?? "—"}</span>
        </div>
      </div>

      {(ev.normalized_record_ids?.length ?? 0) > 0 ? (
        <div className="mt-3">
          <div className="text-xs font-medium text-neutral-800">Normalized records</div>
          <ul className="mt-1 space-y-1 text-xs">
            {(ev.normalized_record_ids ?? []).map((nid) => (
              <li key={nid}>
                <Link href={`/normalized-records/${nid}`} className="text-blue-700 underline">
                  {nid}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {(ev.raw_artifact_ids?.length ?? 0) > 0 ? (
        <div className="mt-3">
          <div className="text-xs font-medium text-neutral-800">Raw artifacts</div>
          <ul className="mt-1 space-y-1 text-xs">
            {(ev.raw_artifact_ids ?? []).map((rid) => (
              <li key={rid}>
                <Link href={`/raw-artifacts/${rid}`} className="text-blue-700 underline">
                  {rid}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </li>
  ));
}

export function EntityDetailView({
  detail,
  related,
  linkedExplore,
  evidence,
  contentBlocks,
  onWorkflowPatch,
  workflowBusy,
  workflowErr,
}: {
  detail: EntityDetailResponse;
  related: RelatedEntityItem[] | null;
  /** Explicit `entity_links` traversal (optional `depth=2`); permission-checked server-side. */
  linkedExplore?: RelatedEntityItem[] | null;
  evidence: EntityEvidenceResponse | null;
  contentBlocks?: ContentBlockRow[] | null;
  onWorkflowPatch?: (lifecycle: string) => void;
  workflowBusy?: boolean;
  workflowErr?: string | null;
}) {
  const { entity } = detail;
  const [showFull, setShowFull] = useState(false);
  const [evidenceOpen, setEvidenceOpen] = useState(false);

  const hasPreview = Boolean(detail.content_preview?.text);
  const fullBody = entity.body?.trim() ?? "";
  const showDocSurface = entity.type === "ReferenceDocument" || Boolean(detail.external_ref) || Boolean(fullBody);

  const provenanceByType = useMemo(() => {
    const m = new Map<string, number>();
    for (const p of detail.provenance) m.set(p.origin_type, (m.get(p.origin_type) ?? 0) + 1);
    return Array.from(m.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5);
  }, [detail.provenance]);

  const evidenceByType = useMemo(() => {
    const m = new Map<string, number>();
    for (const e of evidence?.evidence ?? []) m.set(e.record.origin_type, (m.get(e.record.origin_type) ?? 0) + 1);
    return Array.from(m.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5);
  }, [evidence?.evidence]);

  const primaryEvidence = useMemo(() => (evidence?.evidence ?? []).slice(0, 3), [evidence?.evidence]);
  const secondaryEvidence = useMemo(() => (evidence?.evidence ?? []).slice(3), [evidence?.evidence]);

  const life = detail.lifecycle_state;
  const stages = ["draft", "in_review", "published"] as const;
  const stageIndex = stages.indexOf(life as (typeof stages)[number]);
  const activeIdx = stageIndex >= 0 ? stageIndex : 0;

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
      <section className="min-w-0">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <h1 className="truncate text-2xl font-semibold tracking-tight">{entity.title || "Untitled"}</h1>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <TrustBadge truthMode={detail.truth_mode} />
              <span className="rounded-full border border-neutral-200 bg-white px-2 py-0.5 text-xs text-neutral-800">
                {entity.type}
              </span>
              <TrustExplanationDrawer
                meta={{
                  truthMode: detail.truth_mode,
                  lifecycleState: detail.lifecycle_state,
                  freshnessStatus: detail.freshness_status,
                  approvalStatus: detail.approval_status,
                  canonicalStatus: detail.canonical_status,
                  ownerId: detail.owner_id,
                  domainId: detail.domain_id,
                  entityType: entity.type,
                }}
              />
              <span className="text-xs text-neutral-600">
                Snapshot{" "}
                <code className="rounded bg-neutral-100 px-1">{new Date(detail.snapshot_at).toISOString()}</code>
              </span>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            {detail.open_in_source_url ? (
              <a
                href={detail.open_in_source_url}
                target="_blank"
                rel="noreferrer"
                className="rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm text-neutral-900 hover:bg-neutral-50"
              >
                View at source
              </a>
            ) : null}
            {detail.external_ref ? (
              <button
                type="button"
                className="rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm text-neutral-900 hover:bg-neutral-50"
                onClick={() => void navigator.clipboard?.writeText(detail.external_ref ?? "")}
              >
                Copy external ref
              </button>
            ) : null}
          </div>
        </div>

        <div className="mt-6 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium text-neutral-900">Publication workflow</h2>
          <p className="mt-1 text-xs text-neutral-600">
            Stages reflect <code className="rounded bg-neutral-100 px-1">lifecycle_state</code>. Approve via open review tasks in Governance when assigned.
          </p>
          <ol className="mt-3 flex flex-wrap gap-2 text-xs">
            {stages.map((s, i) => (
              <li
                key={s}
                className={`rounded-full border px-2 py-0.5 ${i <= activeIdx ? "border-emerald-300 bg-emerald-50 text-emerald-900" : "border-neutral-200 bg-white text-neutral-600"}`}
              >
                {s}
              </li>
            ))}
          </ol>
          {workflowErr ? <p className="mt-2 text-xs text-red-700">{workflowErr}</p> : null}
          {onWorkflowPatch ? (
            <div className="mt-3 flex flex-wrap gap-2">
              {life === "draft" ? (
                <button
                  type="button"
                  disabled={workflowBusy}
                  className="rounded-md border border-neutral-300 bg-white px-2 py-1 text-xs disabled:opacity-50"
                  onClick={() => onWorkflowPatch("in_review")}
                >
                  Submit for review
                </button>
              ) : null}
              {life === "in_review" ? (
                <button
                  type="button"
                  disabled={workflowBusy}
                  className="rounded-md border border-neutral-300 bg-white px-2 py-1 text-xs disabled:opacity-50"
                  onClick={() => onWorkflowPatch("draft")}
                >
                  Move back to draft
                </button>
              ) : null}
              <Link href="/governance" className="rounded-md border border-neutral-300 bg-white px-2 py-1 text-xs text-neutral-900">
                Open governance
              </Link>
            </div>
          ) : null}
        </div>

        <div className="mt-6 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium text-neutral-900">Metadata</h2>
          <div className="mt-3 grid gap-2">
            <KeyValue k="Entity ID" v={<code className="rounded bg-neutral-100 px-1">{entity.id}</code>} />
            <KeyValue k="Truth mode" v={<span>{formatTruthMode(detail.truth_mode)}</span>} />
            <KeyValue k="Freshness" v={<span>{detail.freshness_status}</span>} />
            <KeyValue k="Source" v={<span>{detail.source ?? "—"}</span>} />
            <KeyValue k="External ref" v={<span className="break-all">{detail.external_ref ?? "—"}</span>} />
            <KeyValue k="Owner" v={<span>{detail.owner_id ?? "—"}</span>} />
            <KeyValue k="Domain" v={<span>{detail.domain_id}</span>} />
            <KeyValue k="Sensitivity" v={<span>{detail.sensitivity_level}</span>} />
            <KeyValue k="Lifecycle" v={<span>{detail.lifecycle_state}</span>} />
          </div>
        </div>

        {showDocSurface ? (
          <div className="mt-6 rounded-lg border border-neutral-200 p-4">
            <h2 className="text-sm font-medium text-neutral-900">Preview</h2>
            <p className="mt-1 text-xs text-neutral-600">
              This is a snapshot view. Trust mode and provenance are shown separately.
            </p>

            {!hasPreview && !fullBody ? (
              <p className="mt-3 text-sm text-neutral-600">No content preview available.</p>
            ) : (
              <div className="mt-3">
                <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap rounded bg-neutral-50 p-3 text-sm text-neutral-900">
                  {showFull && fullBody ? fullBody : detail.content_preview?.text ?? fullBody}
                </pre>
                {detail.content_preview?.truncated && fullBody ? (
                  <button
                    type="button"
                    className="mt-2 text-sm text-blue-700 underline"
                    onClick={() => setShowFull((v) => !v)}
                  >
                    {showFull ? "Show excerpt" : "Show full content"}
                  </button>
                ) : null}
              </div>
            )}
          </div>
        ) : null}

        <div className="mt-6 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium text-neutral-900">Provenance</h2>
          <p className="mt-1 text-xs text-neutral-600">
            Evidence is shown as records. Raw artifacts remain permission-scoped elsewhere.
          </p>

          {detail.provenance.length === 0 ? (
            <p className="mt-3 text-sm text-neutral-600">No provenance records.</p>
          ) : (
            <>
              <div className="mt-3 flex flex-wrap gap-2 text-xs text-neutral-700">
                {provenanceByType.map(([t, n]) => (
                  <span key={t} className="rounded-full border border-neutral-200 bg-white px-2 py-0.5">
                    {t} · {n}
                  </span>
                ))}
              </div>
              <ul className="mt-3 space-y-2 text-sm">
                {detail.provenance.map((p) => (
                  <li key={p.id} className="rounded border border-neutral-200 p-3">
                    <div className="flex flex-wrap items-baseline justify-between gap-2">
                      <div className="font-medium text-neutral-900">{p.origin_type}</div>
                      <div className="text-xs text-neutral-600">{new Date(p.created_at).toISOString()}</div>
                    </div>
                    <div className="mt-2 grid gap-1 text-xs text-neutral-700">
                      <div>
                        origin_ref: <span className="break-all">{p.origin_ref ?? "—"}</span>
                      </div>
                      <div>
                        source_feed_id: <span className="break-all">{p.source_feed_id ?? "—"}</span>
                      </div>
                      <div>
                        job_run_id: <span className="break-all">{p.job_run_id ?? "—"}</span>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>

        {contentBlocks && contentBlocks.length > 0 ? (
          <div className="mt-6 rounded-lg border border-neutral-200 p-4">
            <h2 className="text-sm font-medium text-neutral-900">Content blocks</h2>
            <p className="mt-1 text-xs text-neutral-600">Reusable blocks attached to this entity (governed separately).</p>
            <ul className="mt-3 space-y-3">
              {contentBlocks.map((b) => (
                <li key={b.id} className="rounded border border-neutral-200 p-3 text-sm">
                  <div className="font-medium text-neutral-900">{b.title}</div>
                  <div className="mt-1 text-xs text-neutral-600">
                    {b.truth_mode} · {b.lifecycle_state} · {b.approval_status}
                  </div>
                  <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-neutral-50 p-2 text-xs">{b.body}</pre>
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        <div className="mt-6 rounded-lg border border-neutral-200 p-4">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h2 className="text-sm font-medium text-neutral-900">Evidence</h2>
            <button
              type="button"
              className="text-sm text-blue-700 underline"
              onClick={() => setEvidenceOpen((v) => !v)}
            >
              {evidenceOpen ? "Collapse" : "Inspect evidence"}
            </button>
          </div>
          <p className="mt-1 text-xs text-neutral-600">
            Links to raw artifacts and normalized records appear only if you have permission. No hidden access broadening.
          </p>

          {!evidence ? (
            <p className="mt-3 text-sm text-neutral-600">No evidence loaded.</p>
          ) : evidence.evidence.length === 0 ? (
            <p className="mt-3 text-sm text-neutral-600">No evidence records.</p>
          ) : (
            <>
              <div className="mt-3 flex flex-wrap gap-2 text-xs text-neutral-700">
                {evidenceByType.map(([t, n]) => (
                  <span key={t} className="rounded-full border border-neutral-200 bg-white px-2 py-0.5">
                    {t} · {n}
                  </span>
                ))}
              </div>
              <div className="mt-3 text-xs text-neutral-600">
                raw_access={String(evidence.can_view_raw)}; normalized_access={String(evidence.can_view_normalized)}
              </div>
              {evidenceOpen ? (
                <div className="mt-3 space-y-4 text-sm">
                  <div>
                    <div className="text-xs font-medium text-neutral-800">Primary (direct material)</div>
                    <ul className="mt-2 space-y-2">{renderEvidenceItems(primaryEvidence)}</ul>
                  </div>
                  {secondaryEvidence.length > 0 ? (
                    <div>
                      <div className="text-xs font-medium text-neutral-800">Secondary (additional provenance)</div>
                      <ul className="mt-2 space-y-2">{renderEvidenceItems(secondaryEvidence)}</ul>
                    </div>
                  ) : null}
                </div>
              ) : null}
            </>
          )}
        </div>
      </section>

      <aside className="min-w-0 space-y-4">
        <div className="rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium text-neutral-900">Explore from here</h2>
          <p className="mt-1 text-xs text-neutral-600">
            Governed <code className="rounded bg-neutral-100 px-1">entity_links</code> only (1–2 hops). Items you cannot view are omitted; no existence leak.
          </p>

          {!linkedExplore || linkedExplore.length === 0 ? (
            <p className="mt-3 text-sm text-neutral-600">No linked entities in your scope.</p>
          ) : (
            <ul className="mt-3 space-y-2">
              {linkedExplore.map((r) => (
                <li key={r.entity.id} className="rounded border border-neutral-200 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <Link href={`/entities/${r.entity.id}`} className="min-w-0 truncate text-sm text-blue-700 underline">
                      {r.entity.title || r.entity.id}
                    </Link>
                    <TrustBadge truthMode={r.entity.truth_mode} />
                  </div>
                  <div className="mt-2 text-xs text-neutral-700">
                    <div>type: {r.entity.type}</div>
                    <div>freshness: {r.entity.freshness_status}</div>
                    <div className="text-neutral-600">reason: {r.reason}</div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium text-neutral-900">Recommendations</h2>
          <p className="mt-1 text-xs text-neutral-600">
            Explainable picks: explicit links, curated hub neighbors, then trusted same-domain content. Each item is access-checked.
          </p>

          {!related || related.length === 0 ? (
            <p className="mt-3 text-sm text-neutral-600">No related content.</p>
          ) : (
            <ul className="mt-3 space-y-2">
              {related.map((r) => (
                <li key={r.entity.id} className="rounded border border-neutral-200 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <Link href={`/entities/${r.entity.id}`} className="min-w-0 truncate text-sm text-blue-700 underline">
                      {r.entity.title || r.entity.id}
                    </Link>
                    <TrustBadge truthMode={r.entity.truth_mode} />
                  </div>
                  <div className="mt-2 text-xs text-neutral-700">
                    <div>type: {r.entity.type}</div>
                    <div>freshness: {r.entity.freshness_status}</div>
                    <div className="text-neutral-600">reason: {r.reason}</div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </aside>
    </div>
  );
}

