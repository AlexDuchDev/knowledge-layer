"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { KnowledgeCard } from "@/components/KnowledgeCard";
import { FollowScopeButton } from "@/components/FollowScopeButton";
import { PartialViewNotice } from "@/components/PartialViewNotice";
import { WorkflowNextSteps } from "@/components/WorkflowNextSteps";
import { OnboardingChecklist } from "@/components/onboarding/OnboardingChecklist";
import { useAuthSession } from "@/hooks/useAuthSession";
import { apiBase, apiJson, principalUserId, isDevPrincipalHeader } from "@/lib/api";
import type { InstanceStatus } from "@/lib/instanceStatus";
import { canAccessZoneSwitcher, canSeeInstanceSetupChecklist } from "@/lib/navigation";

type Ent = {
  id: string;
  type: string;
  title: string;
  truth_mode: string;
  lifecycle_state: string;
  freshness_status: string;
  domain_id?: string;
};

type ReviewPrev = {
  id: string;
  target_id: string;
  status: string;
  title?: string;
  entity_type?: string;
};

type Feed = {
  can_publish: boolean;
  has_reviewer_surface: boolean;
  important_decisions: Ent[];
  recent_approved_content: Ent[];
  featured_collections: { message: string; hubs_url: string } | null;
  recent_digests: Ent[];
  pending_reviews?: ReviewPrev[];
  recommended_reads?: { entity: Ent; reason: string }[];
  from_followed_scopes?: Ent[];
  recent_in_your_work?: Ent[];
};

function Section({
  title,
  subtitle,
  className,
  children,
}: {
  title: string;
  subtitle?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <section className={`rounded-xl border border-neutral-200 bg-white p-4 shadow-sm${className ? ` ${className}` : ""}`}>
      <h2 className="text-sm font-semibold text-neutral-900">{title}</h2>
      {subtitle ? <p className="mt-1 text-xs text-neutral-600">{subtitle}</p> : null}
      <div className="mt-3">{children}</div>
    </section>
  );
}

export default function Home() {
  const { navVis } = useAuthSession();
  const showOperatorHome = canAccessZoneSwitcher(navVis);
  const showInstanceSetupPanel = canSeeInstanceSetupChecklist(navVis);

  const [feed, setFeed] = useState<Feed | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(true);
  const [inst, setInst] = useState<InstanceStatus | null>(null);
  const [checklistOpen, setChecklistOpen] = useState(false);
  const [checklistPermDismissed, setChecklistPermDismissed] = useState(false);
  const [devDetailsOpen, setDevDetailsOpen] = useState(false);

  const load = useCallback(async () => {
    setBusy(true);
    setErr(null);
    try {
      setFeed(await apiJson<Feed>("/home/feed"));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setFeed(null);
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    setChecklistPermDismissed(localStorage.getItem("kl_admin_checklist_dismissed") === "1");
    (async () => {
      try {
        setInst(await apiJson<InstanceStatus>("/instance/status"));
      } catch {
        setInst(null);
      }
    })();
  }, []);

  const mapEnt = (e: Ent) => (
    <KnowledgeCard
      key={e.id}
      variant="entity"
      density="compact"
      title={e.title}
      href={`/entities/${e.id}`}
      entityType={e.type}
      truthMode={e.truth_mode}
      lifecycleState={e.lifecycle_state}
      freshnessStatus={e.freshness_status}
    />
  );

  return (
    <main className="min-h-screen bg-neutral-50 px-4 py-10 sm:px-6">
      <div className="mx-auto max-w-5xl">
        <header className="mb-8">
          <h1 className="text-2xl font-semibold tracking-tight text-neutral-900">Knowledge Layer</h1>
          <p className="mt-2 max-w-2xl text-sm text-neutral-600">
            {showOperatorHome ? (
              <>
                Governed organizational memory: permission-aware search and Ask, entity detail, and operator tools in the control plane. This local build is
                honest about what is wired end-to-end versus preview-only.
              </>
            ) : (
              <>
                Search and Ask over content you are allowed to see, open items from results for full detail, and use the sidebar for more areas your account can
                access.
              </>
            )}
          </p>
          <div className="mt-4 rounded-lg border border-emerald-200 bg-emerald-50/80 p-4 text-sm text-neutral-800">
            <p className="font-medium text-neutral-900">Local golden path</p>
            <ol className="mt-2 list-decimal space-y-1 pl-5 text-xs text-neutral-700">
              <li>
                {inst === null ? (
                  <>Checking instance status…</>
                ) : inst.needs_bootstrap ? (
                  <Link href="/bootstrap" className="text-blue-800 underline">
                    Bootstrap the instance
                  </Link>
                ) : (
                  <>Workspace is present — continue below.</>
                )}
              </li>
              <li>
                Use <Link href="/search" className="text-blue-800 underline">Search</Link>, open a result, then try <Link href="/ask" className="text-blue-800 underline">Ask</Link> for cited answers.
              </li>
              {showOperatorHome ? (
                <li>Operators: configure sources and jobs from the control plane (ingestion depth varies by connector—see docs).</li>
              ) : navVis?.may_manage_source_feed ? (
                <li>
                  Connect data: use the{" "}
                  <Link href="/source-feeds?from=cp" className="text-blue-800 underline">
                    source feed wizard
                  </Link>{" "}
                  and the{" "}
                  <Link href="/control-plane/sources" className="text-blue-800 underline">
                    Sources hub
                  </Link>
                  .
                </li>
              ) : null}
            </ol>
          </div>
          <div className="mt-3">
            <button
              type="button"
              className="text-xs text-neutral-500 underline decoration-dotted hover:text-neutral-700"
              onClick={() => setDevDetailsOpen((o) => !o)}
              aria-expanded={devDetailsOpen}
            >
              {devDetailsOpen ? "Hide" : "Show"} developer details
            </button>
            {devDetailsOpen ? (
              <p className="mt-2 text-xs text-neutral-500">
                API <code className="rounded bg-neutral-100 px-1">{apiBase()}</code>
                {isDevPrincipalHeader() ? (
                  <>
                    {" "}
                    · dev principal <code className="rounded bg-neutral-100 px-1">{principalUserId()}</code>
                  </>
                ) : (
                  <> · session auth (sign in via Login)</>
                )}
              </p>
            ) : null}
          </div>
        </header>

        {showInstanceSetupPanel ? <OnboardingChecklist /> : null}

        {inst?.needs_bootstrap ? (
          <div className="mb-6 rounded-md border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-950">
            This instance has no workspace yet.{" "}
            <Link href="/bootstrap" className="font-medium text-blue-800 underline">
              Run first-time setup
            </Link>
            {showOperatorHome ? (
              <>
                {" · "}
                <Link href="/control-plane/setup" className="font-medium text-blue-800 underline">
                  Control plane setup hub
                </Link>
              </>
            ) : null}
            .
          </div>
        ) : inst && !inst.needs_bootstrap && showOperatorHome ? (
          <div className="mb-6 rounded-md border border-neutral-200 bg-white px-4 py-2 text-sm text-neutral-700">
            <span className="text-neutral-500">Operators:</span>{" "}
            <Link href="/control-plane/governance" className="font-medium text-blue-800 underline">
              Open control plane
            </Link>
            {" · "}
            <Link href="/ask" className="font-medium text-blue-800 underline">
              Ask
            </Link>{" "}
            (same product shell as Home—there is no separate “app” URL).
          </div>
        ) : null}

        {showOperatorHome ? (
          <div className="mb-6 rounded-lg border border-neutral-200 bg-white p-4 text-sm text-neutral-800">
            <p className="font-medium text-neutral-900">What works locally right now</p>
            <ul className="mt-2 list-disc space-y-1 pl-5 text-xs text-neutral-700">
              <li>Search and Ask over entities you can read; results respect domain grants.</li>
              <li>
                Knowledge job runs: only the <code className="rounded bg-neutral-100 px-1">weekly_digest</code> processor is fully implemented; other job types
                fail with a clear error if run.
              </li>
              <li>Connector ingestion: Telegram-style normalization is the main path; other artifact types may no-op until a normalizer exists.</li>
              <li>OpenSearch in local compose is dev-oriented (no security); search depends on it being up.</li>
            </ul>
          </div>
        ) : null}

        {checklistOpen ? (
          <div className="mb-6 rounded-lg border border-blue-200 bg-blue-50/80 p-4 text-sm text-neutral-800">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <h2 className="font-semibold text-neutral-900">Administrator checklist</h2>
                <ol className="mt-2 list-decimal space-y-1 pl-5 text-xs text-neutral-700">
                  <li>
                    <Link href="/bootstrap" className="text-blue-800 underline">
                      Bootstrap
                    </Link>{" "}
                    if needed, then{" "}
                    <Link href="/control-plane/users" className="text-blue-800 underline">
                      users and access
                    </Link>
                    .
                  </li>
                  <li>
                    <Link href="/control-plane/sources/connectors" className="text-blue-800 underline">
                      Connectors
                    </Link>{" "}
                    →{" "}
                    <Link href="/control-plane/sources" className="text-blue-800 underline">
                      Sources
                    </Link>{" "}
                    (end-to-end depth varies by type).
                  </li>
                  <li>
                    <Link href="/settings" className="text-blue-800 underline">
                      Instance settings
                    </Link>{" "}
                    when you need SMTP or env reference.
                  </li>
                </ol>
              </div>
              <button
                type="button"
                className="shrink-0 rounded border border-neutral-300 bg-white px-2 py-1 text-xs"
                onClick={() => {
                  localStorage.setItem("kl_admin_checklist_dismissed", "1");
                  setChecklistPermDismissed(true);
                  setChecklistOpen(false);
                }}
              >
                Dismiss
              </button>
            </div>
          </div>
        ) : showOperatorHome && !checklistPermDismissed ? (
          <div className="mb-6">
            <button type="button" className="text-xs text-blue-800 underline" onClick={() => setChecklistOpen(true)}>
              Show administrator checklist
            </button>
          </div>
        ) : null}

        {err ? <div className="mb-6 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}

        <div className="mb-6">
          <PartialViewNotice detailed={showOperatorHome} />
        </div>
        <div className="mb-6 rounded-md border border-neutral-200 bg-white px-3 py-2">
          <WorkflowNextSteps variant={showOperatorHome ? "operator" : "default"} />
        </div>

        <div className="mb-6 flex flex-wrap items-center gap-3">
          <Link href="/search" className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white">
            Search
          </Link>
          <Link href="/ask" className="rounded-md border border-neutral-300 bg-white px-4 py-2 text-sm font-medium text-neutral-900">
            Ask
          </Link>
          <span className="text-xs text-neutral-500">
            More routes live in the sidebar: browse types, governance, control plane, and advanced tools.
          </span>
        </div>

        {busy && !feed ? <p className="text-sm text-neutral-600">Loading your feed…</p> : null}

        {feed ? (
          <div className="grid gap-4 lg:grid-cols-2">
            {feed.recommended_reads && feed.recommended_reads.length > 0 ? (
              <Section
                title="Suggested reads"
                subtitle="Published and approved, ranked by freshness. Reasons are explicit (not a black-box score)."
                className="lg:col-span-2"
              >
                <div className="grid gap-2 sm:grid-cols-2">
                  {feed.recommended_reads.map((row) => (
                    <div key={row.entity.id} className="rounded-lg border border-neutral-200 bg-neutral-50/50 p-2">
                      {mapEnt(row.entity)}
                      <p className="mt-1 text-[10px] text-neutral-500">{row.reason}</p>
                    </div>
                  ))}
                </div>
              </Section>
            ) : null}

            {feed.from_followed_scopes && feed.from_followed_scopes.length > 0 ? (
              <Section
                title="From scopes you follow"
                subtitle="Published content from domains, topic hubs, or topic slices you follow. Following only affects surfacing, not access."
                className="lg:col-span-2"
              >
                <div className="grid gap-2 sm:grid-cols-2">{feed.from_followed_scopes.map(mapEnt)}</div>
              </Section>
            ) : null}

            {feed.recent_in_your_work && feed.recent_in_your_work.length > 0 ? (
              <Section
                title="Recently updated in your domains"
                subtitle="Lightweight activity signal — items you can view, ordered by update time."
                className="lg:col-span-2"
              >
                <div className="grid gap-2 sm:grid-cols-2">{feed.recent_in_your_work.map(mapEnt)}</div>
              </Section>
            ) : null}

            <Section title="Important decisions" subtitle="Recent decision entities in domains you can access.">
              {(feed.important_decisions ?? []).length === 0 ? (
                <p className="text-sm text-neutral-600">No decisions yet.</p>
              ) : (
                <div className="space-y-2">{(feed.important_decisions ?? []).map(mapEnt)}</div>
              )}
            </Section>

            <Section title="Recently approved" subtitle="Published content after governance approval.">
              {(feed.recent_approved_content ?? []).length === 0 ? (
                <p className="text-sm text-neutral-600">No recent approved items.</p>
              ) : (
                <div className="space-y-2">{(feed.recent_approved_content ?? []).map(mapEnt)}</div>
              )}
            </Section>

            <Section title="Featured collections" subtitle={feed.featured_collections?.message ?? ""}>
              <Link href={feed.featured_collections?.hubs_url ?? "/hubs"} className="text-sm text-blue-700 underline">
                Open topic hubs
              </Link>
            </Section>

            <Section title="Recent digests" subtitle="Digest-style insights when present.">
              {(feed.recent_digests ?? []).length === 0 ? (
                <p className="text-sm text-neutral-600">No digests yet.</p>
              ) : (
                <>
                  <div className="mb-3 flex flex-wrap gap-4 border-b border-neutral-100 pb-3">
                    <p className="w-full text-xs text-neutral-600">Digest delivery (in-app): follow a domain digest stream to prioritize it on Home.</p>
                    {Array.from(new Set((feed.recent_digests ?? []).map((e) => e.domain_id).filter(Boolean))).map((did) => (
                      <FollowScopeButton key={did} scopeType="digest_stream" refId={did!} className="min-w-[200px]" />
                    ))}
                  </div>
                  <div className="space-y-2">{(feed.recent_digests ?? []).map(mapEnt)}</div>
                </>
              )}
            </Section>

            {feed.pending_reviews && feed.pending_reviews.length > 0 ? (
              <Section title="Pending reviews" subtitle={feed.can_publish ? "Open tasks in your governance scope." : "Assigned to you or owned by you."}>
                <ul className="space-y-2 text-sm">
                  {feed.pending_reviews.map((r) => (
                    <li key={r.id} className="rounded border border-amber-200 bg-amber-50 px-3 py-2">
                      <Link href={`/governance`} className="font-medium text-blue-800 underline">
                        {r.title || r.target_id}
                      </Link>
                      <div className="text-xs text-neutral-600">
                        {r.entity_type ?? "entity"} · {r.status}
                      </div>
                    </li>
                  ))}
                </ul>
              </Section>
            ) : null}
          </div>
        ) : null}
      </div>
    </main>
  );
}
