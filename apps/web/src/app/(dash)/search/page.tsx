"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { WorkflowNextSteps } from "@/components/WorkflowNextSteps";
import { PartialViewNotice } from "@/components/PartialViewNotice";
import { KnowledgeCard } from "@/components/KnowledgeCard";
import { TrustExplanationDrawer } from "@/components/TrustExplanationDrawer";
import { apiJson } from "@/lib/api";
import { DocHelpCallout } from "@/components/guidance/DocHelpCallout";
import { BROWSE_ROUTES } from "@/lib/entityTypes";
import { SEARCH_PRESETS, type SearchPresetId, presetById } from "@/lib/searchPresets";

type SearchHit = {
  entity_id: string;
  domain_id: string;
  domain_name?: string;
  owner_id?: string;
  owner_name?: string;
  entity_type: string;
  title: string;
  truth_mode: string;
  lifecycle_state: string;
  freshness_status: string;
  trust_summary: string;
  snippet?: string;
  relation_expansion?: string;
};

const SEARCH_PLACEHOLDERS = [
  "Search decisions, policies, meetings, and insights",
  "Find the latest approved onboarding SOP",
  "Search finance decisions from last quarter",
];

type SearchResponse = { hits: SearchHit[] };

type DomainRow = { id: string; name: string };

function qp(params: Record<string, string | undefined>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    const vv = (v ?? "").trim();
    if (vv) sp.set(k, vv);
  }
  const s = sp.toString();
  return s ? `?${s}` : "";
}

function SearchPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [hits, setHits] = useState<SearchHit[] | null>(null);
  const [domains, setDomains] = useState<DomainRow[] | null>(null);

  const [q, setQ] = useState(() => searchParams.get("q") ?? "");
  const [domainId, setDomainId] = useState(() => searchParams.get("domain_id") ?? "");
  const [ownerId, setOwnerId] = useState(() => searchParams.get("owner_id") ?? "");
  const [type, setType] = useState(() => searchParams.get("type") ?? "");
  const [truthMode, setTruthMode] = useState(() => searchParams.get("truth_mode") ?? "");
  const [lifecycleState, setLifecycleState] = useState(() => searchParams.get("lifecycle_state") ?? "");
  const [freshnessStatus, setFreshnessStatus] = useState(() => searchParams.get("freshness_status") ?? "");
  const [approvalStatus, setApprovalStatus] = useState(() => searchParams.get("approval_status") ?? "");
  const [expandRelations, setExpandRelations] = useState(() => searchParams.get("expand_relations") === "1");

  const activePreset = (searchParams.get("preset") as SearchPresetId | null) || undefined;

  const searchPlaceholder = useRef(SEARCH_PLACEHOLDERS[Math.floor(Math.random() * SEARCH_PLACEHOLDERS.length)]).current;

  const scopeSummaryLines = useMemo(() => {
    const lines: string[] = [
      "Results are limited to domains (and sensitivity) your account can read—they are not a search of the whole organization.",
    ];
    const dname = domains?.find((d) => d.id === domainId)?.name;
    if (domainId && dname) lines.push(`Domain filter: ${dname}.`);
    else if (domainId) lines.push(`Domain filter: ${domainId}.`);
    if (ownerId.trim()) lines.push(`Owner filter: ${ownerId.trim()}.`);
    if (type.trim()) lines.push(`Type: ${type.trim()}.`);
    if (truthMode.trim()) lines.push(`Truth mode: ${truthMode.trim()}.`);
    if (lifecycleState.trim()) lines.push(`Lifecycle: ${lifecycleState.trim()}.`);
    if (freshnessStatus.trim()) lines.push(`Freshness: ${freshnessStatus.trim()}.`);
    if (approvalStatus.trim()) lines.push(`Approval: ${approvalStatus.trim()}.`);
    if (expandRelations) lines.push("Related expansion: 1-hop entity links in the same granted domains.");
    return lines;
  }, [domains, domainId, ownerId, type, truthMode, lifecycleState, freshnessStatus, approvalStatus, expandRelations]);

  const weakResultHint = useMemo(() => {
    if (!hits || !q.trim()) return null;
    if (hits.length === 0) return null;
    const derivedish = hits.filter((h) => h.truth_mode.toLowerCase().includes("derived")).length;
    const staleish = hits.filter((h) => h.freshness_status.toLowerCase() === "stale").length;
    if (hits.length <= 3) {
      return "Limited relevant results in this scope—try removing a filter, broadening the query, or use Ask for synthesis across permitted knowledge.";
    }
    if (derivedish >= Math.ceil(hits.length * 0.6)) {
      return "Many hits are derived-class; consider narrowing to canonical or approved sources.";
    }
    if (staleish >= Math.ceil(hits.length * 0.5)) {
      return "Several results look stale; verify freshness before relying on them.";
    }
    return null;
  }, [hits, q]);

  useEffect(() => {
    const p = searchParams.get("preset");
    if (!p || p === "all") return;
    if (p === "my_domain") return;
    const def = presetById(p);
    if (!def) return;
    setType(def.params.type ?? "");
    setLifecycleState(def.params.lifecycle_state ?? "");
    setTruthMode(def.params.truth_mode ?? "");
  }, [searchParams]);

  useEffect(() => {
    void (async () => {
      try {
        const d = await apiJson<DomainRow[]>("/domains");
        setDomains(d);
      } catch {
        setDomains([]);
      }
    })();
  }, []);

  useEffect(() => {
    if (activePreset === "my_domain" && domains && domains.length > 0 && !domainId) {
      setDomainId(domains[0].id);
    }
  }, [activePreset, domains, domainId]);

  const query = useMemo(
    () =>
      qp({
        q,
        domain_id: domainId,
        owner_id: ownerId,
        type,
        truth_mode: truthMode,
        lifecycle_state: lifecycleState,
        freshness_status: freshnessStatus,
        approval_status: approvalStatus,
        expand_relations: expandRelations ? "1" : "",
      }),
    [q, domainId, ownerId, type, truthMode, lifecycleState, freshnessStatus, approvalStatus, expandRelations],
  );

  const syncUrl = useCallback(() => {
    const sp = new URLSearchParams();
    if (q.trim()) sp.set("q", q.trim());
    if (domainId.trim()) sp.set("domain_id", domainId.trim());
    if (ownerId.trim()) sp.set("owner_id", ownerId.trim());
    if (type.trim()) sp.set("type", type.trim());
    if (truthMode.trim()) sp.set("truth_mode", truthMode.trim());
    if (lifecycleState.trim()) sp.set("lifecycle_state", lifecycleState.trim());
    if (freshnessStatus.trim()) sp.set("freshness_status", freshnessStatus.trim());
    if (approvalStatus.trim()) sp.set("approval_status", approvalStatus.trim());
    if (expandRelations) sp.set("expand_relations", "1");
    if (activePreset && activePreset !== "all") sp.set("preset", activePreset);
    const s = sp.toString();
    router.replace(`/search${s ? `?${s}` : ""}`, { scroll: false });
  }, [q, domainId, ownerId, type, truthMode, lifecycleState, freshnessStatus, approvalStatus, expandRelations, activePreset, router]);

  const run = useCallback(async () => {
    setErr(null);
    setBusy(true);
    try {
      const out = await apiJson<SearchResponse>(`/search${query}`);
      setHits(out.hits);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setHits(null);
    } finally {
      setBusy(false);
    }
  }, [query]);

  const applyPreset = (id: SearchPresetId) => {
    if (id === "all") {
      setType("");
      setLifecycleState("");
      setTruthMode("");
      router.replace("/search");
      return;
    }
    if (id === "my_domain") {
      const first = domains?.[0]?.id ?? "";
      setDomainId(first);
      setType("");
      setLifecycleState("");
      setTruthMode("");
      router.replace(`/search?preset=my_domain&domain_id=${encodeURIComponent(first)}`);
      return;
    }
    const def = SEARCH_PRESETS.find((x) => x.id === id);
    if (!def) return;
    setType(def.params.type ?? "");
    setLifecycleState(def.params.lifecycle_state ?? "");
    setTruthMode(def.params.truth_mode ?? "");
    const sp = new URLSearchParams();
    sp.set("preset", id);
    if (def.params.type) sp.set("type", def.params.type);
    if (def.params.lifecycle_state) sp.set("lifecycle_state", def.params.lifecycle_state);
    if (def.params.truth_mode) sp.set("truth_mode", def.params.truth_mode);
    router.replace(`/search?${sp.toString()}`);
  };

  const reportIssue = useCallback(async (entityId: string) => {
    setErr(null);
    try {
      await apiJson("/answer-feedback", {
        method: "POST",
        body: JSON.stringify({
          trace_id: `search:${entityId}`,
          feedback_kind: "weak_citations",
          comment: "pilot: search result issue",
        }),
      });
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, []);

  const presetDescription = presetById(activePreset)?.description ?? "Choose a scope or run a keyword search.";

  return (
    <main className="mx-auto max-w-6xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Search" }]} />
      <div className="mb-4 rounded-md border border-neutral-200 bg-white px-3 py-2">
        <WorkflowNextSteps />
      </div>
      <div className="mb-4">
        <PartialViewNotice />
      </div>
      <p className="mb-4 rounded-md border border-neutral-200 bg-neutral-50 px-3 py-2 text-xs text-neutral-700">
        Local search is backed by <strong>OpenSearch</strong> (see Docker Compose). If the search service is not running, queries will fail until it is
        healthy—this is infrastructure, not missing permissions.
      </p>

      <DocHelpCallout slug="search" />

      <div className="flex flex-col gap-6 lg:flex-row lg:items-start">
        <aside className="w-full shrink-0 rounded-lg border border-neutral-200 bg-neutral-50 p-4 lg:max-w-xs">
          <h2 className="text-sm font-semibold text-neutral-900">Taxonomy</h2>
          <p className="mt-1 text-xs text-neutral-600">
            Browse by knowledge type (product IA §7) or apply the same type as a Search filter.
          </p>
          <ul className="mt-3 space-y-2 text-sm">
            {Object.entries(BROWSE_ROUTES).map(([etype, meta]) => (
              <li key={etype} className="flex flex-wrap items-baseline gap-2">
                <Link href={meta.path} className="text-blue-800 underline hover:text-blue-950">
                  {meta.label}
                </Link>
                <button
                  type="button"
                  className="text-xs text-neutral-600 underline"
                  onClick={() => {
                    setType(etype);
                    const sp = new URLSearchParams();
                    sp.set("type", etype);
                    router.replace(`/search?${sp.toString()}`);
                  }}
                >
                  Filter here
                </button>
              </li>
            ))}
          </ul>
        </aside>

        <div className="min-w-0 flex-1">
      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Search</h1>
          <p className="mt-1 text-sm text-neutral-600">
            Keyword and filter search over entities you are allowed to read. Results are scoped by your domain grants and sensitivity cap—not the whole
            organization.
          </p>
          <ul className="mt-2 list-disc space-y-0.5 pl-5 text-xs text-neutral-600">
            {scopeSummaryLines.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        </div>
        <div className="flex flex-wrap gap-3 text-sm">
          <Link href="/" className="text-blue-700 underline">
            Home
          </Link>
          <Link href="/ask" className="text-blue-700 underline">
            Ask
          </Link>
          <Link href="/governance" className="text-blue-700 underline">
            Governance
          </Link>
        </div>
      </div>

      <section className="mb-4">
        <div className="text-xs font-medium text-neutral-700">Presets</div>
        <div className="mt-2 flex flex-wrap gap-2">
          {SEARCH_PRESETS.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => applyPreset(p.id)}
              className={`rounded-full border px-3 py-1 text-xs ${
                activePreset === p.id ? "border-neutral-900 bg-neutral-900 text-white" : "border-neutral-200 bg-white text-neutral-800"
              }`}
            >
              {p.label}
            </button>
          ))}
          <button
            type="button"
            onClick={() => applyPreset("my_domain")}
            className={`rounded-full border px-3 py-1 text-xs ${
              activePreset === "my_domain" ? "border-neutral-900 bg-neutral-900 text-white" : "border-neutral-200 bg-white text-neutral-800"
            }`}
          >
            My domain
          </button>
        </div>
        <p className="mt-2 text-xs text-neutral-600">{presetDescription}</p>
      </section>

      {err ? (
        <div className="mb-6 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div>
      ) : null}

      <section className="rounded-lg border border-neutral-200 p-4">
        <div className="mb-3">
          <label className="mb-1 block text-xs font-medium text-neutral-700">Search query</label>
          <input
            className="w-full rounded border px-2 py-1 text-sm"
            placeholder={searchPlaceholder}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <p className="mt-1 text-[11px] text-neutral-500">OpenSearch is used when configured on the API; otherwise titles are matched in your granted domains.</p>
        </div>
        <div className="grid gap-3 md:grid-cols-3">
          <Field label="domain_id">
            <select className="w-full rounded border px-2 py-1 text-sm" value={domainId} onChange={(e) => setDomainId(e.target.value)}>
              <option value="">Any granted</option>
              {(domains ?? []).map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="owner_id">
            <input className="w-full rounded border px-2 py-1 text-sm" value={ownerId} onChange={(e) => setOwnerId(e.target.value)} />
          </Field>
          <Field label="type">
            <input className="w-full rounded border px-2 py-1 text-sm" value={type} onChange={(e) => setType(e.target.value)} />
          </Field>
          <Field label="truth_mode">
            <input className="w-full rounded border px-2 py-1 text-sm" value={truthMode} onChange={(e) => setTruthMode(e.target.value)} />
          </Field>
          <Field label="lifecycle_state">
            <input
              className="w-full rounded border px-2 py-1 text-sm"
              value={lifecycleState}
              onChange={(e) => setLifecycleState(e.target.value)}
            />
          </Field>
          <Field label="freshness_status">
            <input
              className="w-full rounded border px-2 py-1 text-sm"
              value={freshnessStatus}
              onChange={(e) => setFreshnessStatus(e.target.value)}
            />
          </Field>
          <Field label="approval_status">
            <select
              className="w-full rounded border px-2 py-1 text-sm"
              value={approvalStatus}
              onChange={(e) => setApprovalStatus(e.target.value)}
            >
              <option value="">Any</option>
              <option value="none">none</option>
              <option value="pending">pending</option>
              <option value="approved">approved</option>
              <option value="rejected">rejected</option>
            </select>
          </Field>
        </div>

        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-sm text-neutral-800">
            <input type="checkbox" checked={expandRelations} onChange={(e) => setExpandRelations(e.target.checked)} />
            Expand relations (1-hop)
          </label>
          <div className="flex items-center gap-2">
            <button type="button" className="rounded border px-2 py-1 text-xs text-neutral-700" onClick={syncUrl}>
              Sync URL
            </button>
            <code className="rounded bg-neutral-100 px-2 py-1 text-xs">{`/search${query}`}</code>
            <button
              type="button"
              className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
              disabled={busy}
              onClick={run}
            >
              {busy ? "Searching…" : "Search"}
            </button>
          </div>
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-lg font-medium">Results</h2>
        <p className="mt-1 text-xs text-neutral-600">Ranked with trust-aware ordering (canonical / published / freshness), then relation expansions deprioritized.</p>
        {!hits ? <p className="mt-2 text-sm text-neutral-600">Run a search to see hits.</p> : null}
        {hits && hits.length === 0 ? (
          <div className="mt-4 rounded-lg border border-neutral-200 bg-neutral-50 px-4 py-3 text-sm text-neutral-800">
            <p className="font-medium">No results matched this query in your current scope.</p>
            <ul className="mt-2 list-disc space-y-1 pl-5 text-neutral-700">
              <li>Try removing one or more filters.</li>
              <li>Try a shorter or broader keyword.</li>
              <li>
                Need a summary instead?{" "}
                <Link className="text-blue-700 underline" href="/ask">
                  Ask
                </Link>{" "}
                synthesizes from permitted search hits (not the whole org).
              </li>
            </ul>
          </div>
        ) : null}
        {weakResultHint ? (
          <div className="mt-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-950">{weakResultHint}</div>
        ) : null}
        {hits && hits.length > 0 ? (
          <div className="mt-4 grid gap-3 md:grid-cols-2">
            {hits.map((h) => (
              <div key={h.entity_id} className="relative">
                <KnowledgeCard
                  variant="entity"
                  density="compact"
                  title={h.title}
                  href={`/entities/${h.entity_id}?from=search&q=${encodeURIComponent(q)}`}
                  entityType={h.entity_type}
                  truthMode={h.truth_mode}
                  lifecycleState={h.lifecycle_state}
                  freshnessStatus={h.freshness_status}
                  snippet={h.snippet || h.trust_summary}
                  footer={
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="min-w-0 text-[10px] text-neutral-600">
                        {h.domain_name ? <span>Domain: {h.domain_name}</span> : null}
                        {h.owner_name ? (
                          <span className={h.domain_name ? " ml-2" : ""}>Owner: {h.owner_name}</span>
                        ) : h.owner_id ? (
                          <span className={h.domain_name ? " ml-2" : ""}>Owner: {h.owner_id.slice(0, 8)}…</span>
                        ) : null}
                        {h.relation_expansion ? <div className="text-neutral-500">{h.relation_expansion}</div> : null}
                      </div>
                      <div className="flex items-center gap-2">
                        <TrustExplanationDrawer
                          meta={{
                            truthMode: h.truth_mode,
                            lifecycleState: h.lifecycle_state,
                            freshnessStatus: h.freshness_status,
                            entityType: h.entity_type,
                            domainId: h.domain_id,
                          }}
                        />
                        <button
                          type="button"
                          className="rounded bg-neutral-900 px-2 py-1 text-[10px] text-white"
                          onClick={() => reportIssue(h.entity_id)}
                        >
                          Report
                        </button>
                      </div>
                    </div>
                  }
                />
              </div>
            ))}
          </div>
        ) : null}
      </section>
        </div>
      </div>
    </main>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 text-xs font-medium text-neutral-700">{label}</div>
      {children}
    </div>
  );
}

export default function SearchPage() {
  return (
    <Suspense fallback={<main className="p-10 text-sm text-neutral-600">Loading search…</main>}>
      <SearchPageInner />
    </Suspense>
  );
}
