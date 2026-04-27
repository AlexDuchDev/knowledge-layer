"use client";

import { useParams, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { apiBase, apiJson, principalUserId } from "@/lib/api";
import { AskPanel } from "@/components/AskPanel";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import {
  EntityDetailView,
  type ContentBlockRow,
  type EntityDetailResponse,
  type RelatedEntityItem,
  type EntityEvidenceResponse,
} from "@/components/EntityDetailView";
import { DraftSuggestionsPanel } from "@/components/DraftSuggestionsPanel";
import { ShareTrustCard } from "@/components/ShareTrustCard";
import { browsePathForEntityType } from "@/lib/knowledgeRoutes";

function EntityDetailPageInner() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const id = params?.id ?? "";
  const from = searchParams.get("from");
  const q = searchParams.get("q");

  const [err, setErr] = useState<string | null>(null);
  const [detail, setDetail] = useState<EntityDetailResponse | null>(null);
  const [linkedExplore, setLinkedExplore] = useState<RelatedEntityItem[] | null>(null);
  const [related, setRelated] = useState<RelatedEntityItem[] | null>(null);
  const [evidence, setEvidence] = useState<EntityEvidenceResponse | null>(null);
  const [versions, setVersions] = useState<unknown[] | null>(null);
  const [blocks, setBlocks] = useState<ContentBlockRow[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [wfBusy, setWfBusy] = useState(false);
  const [wfErr, setWfErr] = useState<string | null>(null);

  const load = useCallback(async () => {
    setErr(null);
    setBusy(true);
    try {
      if (!id) {
        throw new Error("missing id");
      }
      const d = await apiJson<EntityDetailResponse>(`/entities/${id}/detail`);
      const linked = await apiJson<RelatedEntityItem[]>(`/entities/${id}/related?limit=12&depth=2`);
      const r = await apiJson<RelatedEntityItem[]>(`/entities/${id}/recommendations?limit=8`);
      const ev = await apiJson<EntityEvidenceResponse>(`/entities/${id}/evidence`);
      const v = await apiJson<unknown[]>(`/entities/${id}/versions`);
      let bl: ContentBlockRow[] = [];
      try {
        bl = await apiJson<ContentBlockRow[]>(`/entities/${id}/content-blocks`);
      } catch {
        bl = [];
      }
      setDetail(d);
      setLinkedExplore(linked);
      setRelated(r);
      setEvidence(ev);
      setVersions(v);
      setBlocks(bl);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setDetail(null);
      setLinkedExplore(null);
      setRelated(null);
      setEvidence(null);
      setVersions(null);
      setBlocks(null);
    } finally {
      setBusy(false);
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);

  const breadcrumbItems = useMemo(() => {
    const title = detail?.entity.title?.trim() || "Entity";
    if (from === "search") {
      const items = [{ label: "Home", href: "/" }, { label: "Search", href: q ? `/search?q=${encodeURIComponent(q)}` : "/search" }, { label: title }];
      return items;
    }
    const browse = detail ? browsePathForEntityType(detail.entity.type) : null;
    if (browse) {
      return [{ label: "Home", href: "/" }, { label: "Knowledge", href: "/knowledge" }, { label: browse.label, href: browse.href }, { label: title }];
    }
    return [{ label: "Home", href: "/" }, { label: "Entities", href: "/entities" }, { label: title }];
  }, [detail, from, q]);

  const patchLifecycle = useCallback(
    async (lifecycle_state: string) => {
      if (!id) return;
      setWfErr(null);
      setWfBusy(true);
      try {
        await apiJson(`/entities/${id}`, { method: "PATCH", body: JSON.stringify({ lifecycle_state }) });
        await load();
      } catch (e) {
        setWfErr(e instanceof Error ? e.message : String(e));
      } finally {
        setWfBusy(false);
      }
    },
    [id, load],
  );

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <AppBreadcrumb items={breadcrumbItems} />

      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Entity</h1>
          <p className="mt-1 text-sm text-neutral-600">
            Evidence drill-down. API <code className="rounded bg-neutral-100 px-1">{apiBase()}</code> · principal{" "}
            <code className="rounded bg-neutral-100 px-1">{principalUserId()}</code>
          </p>
          <p className="mt-2 text-sm text-neutral-600">
            Entity ID: <code className="rounded bg-neutral-100 px-1">{id}</code>
          </p>
        </div>
        <div className="flex gap-3 text-sm">
          <Link href={from === "search" ? (q ? `/search?q=${encodeURIComponent(q)}` : "/search") : "/search"} className="text-blue-700 underline">
            {from === "search" ? "Back to search" : "Search"}
          </Link>
          <Link href={`/entities/${encodeURIComponent(id)}/explore`} className="text-blue-700 underline" title="Bounded one-hop graph traversal (Phase 2.3.1)">
            Explore from here
          </Link>
          <Link href="/governance" className="text-blue-700 underline">
            Governance
          </Link>
        </div>
      </div>

      {err ? (
        <div className="mb-6 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div>
      ) : null}

      <button
        type="button"
        className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
        disabled={busy}
        onClick={load}
      >
        {busy ? "Refreshing…" : "Refresh"}
      </button>

      {detail ? (
        <section className="mt-6">
          <EntityDetailView
            detail={detail}
            linkedExplore={linkedExplore}
            related={related}
            evidence={evidence}
            contentBlocks={blocks}
            onWorkflowPatch={patchLifecycle}
            workflowBusy={wfBusy}
            workflowErr={wfErr}
          />
          <ShareTrustCard detail={detail} />
          {detail.entity.lifecycle_state === "draft" ? <DraftSuggestionsPanel entityId={id} /> : null}
        </section>
      ) : null}

      <section className="mt-6">
        <h2 className="text-lg font-medium">Versions</h2>
        <pre className="mt-2 max-h-[420px] overflow-auto rounded bg-neutral-50 p-3 text-xs">
          {versions ? JSON.stringify(versions, null, 2) : "—"}
        </pre>
      </section>

      {id ? <AskPanel entityId={id} /> : null}
    </main>
  );
}

export default function EntityDetailPage() {
  return (
    <Suspense fallback={<main className="p-10 text-sm text-neutral-600">Loading entity…</main>}>
      <EntityDetailPageInner />
    </Suspense>
  );
}
