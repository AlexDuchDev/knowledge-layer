"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { EntityBrowseTable, type BrowseEntityRow } from "@/components/EntityBrowseTable";
import { KnowledgeCard } from "@/components/KnowledgeCard";
import { apiJson } from "@/lib/api";
import { ENTITY_TYPES } from "@/lib/entityTypes";
import { KNOWLEDGE_SLUG_TO_TYPE, labelForSlug } from "@/lib/knowledgeRoutes";
import { FollowScopeButton } from "@/components/FollowScopeButton";
import { PartialViewNotice } from "@/components/PartialViewNotice";

type ProcessFocus = "all" | "sop" | "process" | "policy_doc" | "handbook";

function matchesProcessFocus(title: string, f: ProcessFocus): boolean {
  const t = title.toLowerCase();
  if (f === "all") return true;
  if (f === "sop") return /\bsop\b|standard operating|playbook/.test(t);
  if (f === "process") return /\bprocess\b|workflow|procedure/.test(t);
  if (f === "policy_doc") return /\bpolicy\b|policies\b/.test(t);
  if (f === "handbook") return /\bhandbook\b|guide\b|manual\b/.test(t);
  return true;
}

type DomainRow = { id: string; name: string };

export default function KnowledgeTypePage() {
  const params = useParams<{ type: string }>();
  const slug = params?.type ?? "";
  const entityType = KNOWLEDGE_SLUG_TO_TYPE[slug];

  const [domains, setDomains] = useState<DomainRow[] | null>(null);
  const [domainId, setDomainId] = useState("");
  const [rows, setRows] = useState<BrowseEntityRow[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [processFocus, setProcessFocus] = useState<ProcessFocus>("all");
  const [suggested, setSuggested] = useState<{ entity: BrowseEntityRow; reason: string }[] | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setDomains(await apiJson<DomainRow[]>("/domains"));
      } catch {
        setDomains([]);
      }
    })();
  }, []);

  const load = useCallback(async () => {
    if (!entityType) return;
    setErr(null);
    setLoading(true);
    try {
      const q = new URLSearchParams({ limit: "50", offset: "0", type: entityType });
      if (domainId.trim()) q.set("domain_id", domainId.trim());
      const list = await apiJson<BrowseEntityRow[]>(`/entities?${q}`);
      setRows(list);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setRows(null);
    } finally {
      setLoading(false);
    }
  }, [entityType, domainId]);

  useEffect(() => {
    if (entityType) void load();
  }, [entityType, load]);

  useEffect(() => {
    if (!entityType) return;
    void (async () => {
      try {
        const q = new URLSearchParams({ type: entityType, limit: "6" });
        if (domainId.trim()) q.set("domain_id", domainId.trim());
        const rows = await apiJson<{ entity: BrowseEntityRow; reason: string }[]>(`/recommendations/browse?${q}`);
        setSuggested(rows);
      } catch {
        setSuggested(null);
      }
    })();
  }, [entityType, domainId]);

  const filteredRows = useMemo(() => {
    if (!rows) return null;
    if (entityType !== ENTITY_TYPES.process_sop || processFocus === "all") return rows;
    return rows.filter((r) => matchesProcessFocus(r.title ?? "", processFocus));
  }, [rows, entityType, processFocus]);

  if (!entityType) {
    return (
      <main className="p-10">
        <p className="text-sm text-red-700">Unknown browse path.</p>
        <Link href="/knowledge" className="text-blue-700 underline">
          Back
        </Link>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Knowledge", href: "/knowledge" }, { label: labelForSlug(slug) }]} />

      <h1 className="text-2xl font-semibold tracking-tight">{labelForSlug(slug)}</h1>
      <p className="mt-1 text-sm text-neutral-600">Entity type filter: {entityType}</p>

      {err ? <div className="my-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}

      <div className="my-4">
        <PartialViewNotice />
      </div>

      <div className="mt-4 flex flex-wrap items-end gap-4">
        <div>
          <div className="text-xs font-medium text-neutral-700">Domain</div>
          <select className="rounded border px-2 py-1 text-sm" value={domainId} onChange={(e) => setDomainId(e.target.value)}>
            <option value="">All granted</option>
            {(domains ?? []).map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </div>
        {domainId ? <FollowScopeButton scopeType="domain" refId={domainId} className="max-w-[200px]" /> : null}
        {entityType ? (
          <FollowScopeButton
            scopeType="knowledge_topic"
            refId={domainId}
            entityType={entityType}
            className="max-w-xs"
          />
        ) : null}
        <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white" onClick={() => void load()}>
          Refresh
        </button>
      </div>

      {entityType === ENTITY_TYPES.process_sop ? (
        <div className="mt-6">
          <div className="text-xs font-medium text-neutral-700">Process focus (title heuristics)</div>
          <div className="mt-2 flex flex-wrap gap-2">
            {(
              [
                ["all", "All"],
                ["sop", "SOP"],
                ["process", "Process"],
                ["policy_doc", "Policy"],
                ["handbook", "Handbook"],
              ] as const
            ).map(([id, label]) => (
              <button
                key={id}
                type="button"
                onClick={() => setProcessFocus(id)}
                className={`rounded-full border px-3 py-1 text-xs ${
                  processFocus === id ? "border-neutral-900 bg-neutral-900 text-white" : "border-neutral-200 bg-white text-neutral-800"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
          <p className="mt-2 text-xs text-neutral-500">
            Same entity type (<code className="rounded bg-neutral-100 px-1">process_sop</code>); tabs narrow the list by title keywords until subtype metadata ships.
          </p>
        </div>
      ) : null}

      {suggested && suggested.length > 0 ? (
        <section className="mt-8 rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
          <h2 className="text-sm font-semibold text-neutral-900">Suggested in this browse scope</h2>
          <p className="mt-1 text-xs text-neutral-600">
            Published and approved, freshness-ranked. Each item is permission-checked; reason strings are explicit.
          </p>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            {suggested.map((row) => (
              <div key={row.entity.id} className="rounded-lg border border-neutral-100 bg-neutral-50/80 p-2">
                <KnowledgeCard
                  variant="entity"
                  density="compact"
                  title={row.entity.title}
                  href={`/entities/${row.entity.id}`}
                  entityType={row.entity.type}
                  truthMode={row.entity.truth_mode}
                  lifecycleState={row.entity.lifecycle_state}
                  freshnessStatus={row.entity.freshness_status}
                  footer={<span className="text-[10px] text-neutral-500">{row.reason}</span>}
                />
              </div>
            ))}
          </div>
        </section>
      ) : null}

      <div className="mt-6">
        <EntityBrowseTable
          rows={filteredRows}
          loading={loading}
          emptyMessage={
            entityType === ENTITY_TYPES.process_sop && processFocus !== "all"
              ? "No rows match this focus; try All or another tab."
              : "No entities for this type in scope."
          }
        />
      </div>
    </main>
  );
}
