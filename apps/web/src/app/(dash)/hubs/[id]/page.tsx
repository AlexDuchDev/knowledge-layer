"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { KnowledgeCard } from "@/components/KnowledgeCard";
import { apiJson } from "@/lib/api";
import { FollowScopeButton } from "@/components/FollowScopeButton";

type Hub = { id: string; title: string; slug: string; domain_id: string };
type Row = {
  id: string;
  title: string;
  type: string;
  truth_mode: string;
  lifecycle_state: string;
  freshness_status: string;
  hub_role?: string;
};

export default function HubDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<{ hub: Hub; entities: Row[] } | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    void (async () => {
      try {
        setData(await apiJson<{ hub: Hub; entities: Row[] }>(`/content-hubs/${id}/view`));
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setData(null);
      }
    })();
  }, [id]);

  return (
    <main className="mx-auto max-w-4xl px-6 py-10">
      <AppBreadcrumb
        items={[
          { label: "Home", href: "/" },
          { label: "Topic hubs", href: "/hubs" },
          { label: data?.hub.title ?? "Hub" },
        ]}
      />
      {err ? <p className="text-sm text-red-700">{err}</p> : null}
      {data ? (
        <>
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1 className="text-2xl font-semibold">{data.hub.title}</h1>
              <p className="text-xs text-neutral-500">{data.hub.slug}</p>
            </div>
            <FollowScopeButton scopeType="content_hub" refId={data.hub.id} />
          </div>
          <div className="mt-6 grid gap-3 sm:grid-cols-2">
            {data.entities.map((e) => (
              <KnowledgeCard
                key={e.id}
                variant="entity"
                title={e.title}
                href={`/entities/${e.id}`}
                entityType={e.type}
                truthMode={e.truth_mode}
                lifecycleState={e.lifecycle_state}
                freshnessStatus={e.freshness_status}
                footer={<span className="text-[10px] text-neutral-500">Role: {e.hub_role ?? "—"}</span>}
              />
            ))}
          </div>
        </>
      ) : null}
      <p className="mt-8">
        <Link href="/hubs" className="text-sm text-blue-700 underline">
          All hubs
        </Link>
      </p>
    </main>
  );
}
