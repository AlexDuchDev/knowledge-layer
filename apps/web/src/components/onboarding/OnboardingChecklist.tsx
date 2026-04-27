"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson } from "@/lib/api";
import type { InstanceStatus } from "@/lib/instanceStatus";

type Step = { id: string; label: string; done: boolean; href?: string; note?: string };

export function OnboardingChecklist() {
  const [st, setSt] = useState<InstanceStatus | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setSt(await apiJson<InstanceStatus>("/instance/status"));
        setErr(null);
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setSt(null);
      }
    })();
  }, []);

  if (err || !st) {
    return err ? (
      <div className="mb-6 rounded-md border border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-950">Could not load setup status: {err}</div>
    ) : null;
  }

  const totalFeeds = st.source_feed_total_count ?? 0;
  const activeFeeds = st.source_feed_active_count ?? 0;
  const synced = st.has_successful_source_sync ?? false;

  const steps: Step[] = [
    {
      id: "bootstrap",
      label: "Create workspace (bootstrap)",
      done: !st.needs_bootstrap,
      href: "/bootstrap",
      note: st.needs_bootstrap ? "Required before anything else." : undefined,
    },
    {
      id: "source",
      label: "Connect a data source (source feed)",
      done: activeFeeds > 0 || totalFeeds > 0,
      href: "/source-feeds?from=cp",
      note: "Use the guided wizard; activate the feed when ready.",
    },
    {
      id: "sync",
      label: "Run at least one successful sync",
      done: synced,
      href: totalFeeds > 0 ? "/control-plane/sources" : undefined,
      note: "Use feed JSON detail → sync screen, or POST from API. Connector worker must be running in Docker.",
    },
  ];

  const doneCount = steps.filter((s) => s.done).length;

  return (
    <section className="mb-6 rounded-lg border border-blue-200 bg-blue-50/80 p-4 text-sm text-neutral-900">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="font-semibold">Getting started</h2>
        <span className="text-xs text-neutral-600">
          {doneCount}/{steps.length} complete
        </span>
      </div>
      <ol className="mt-3 list-decimal space-y-2 pl-5 text-xs text-neutral-800">
        {steps.map((s) => (
          <li key={s.id} className={s.done ? "text-neutral-500 line-through" : ""}>
            {s.href && !s.done ? (
              <Link href={s.href} className="font-medium text-blue-800 underline">
                {s.label}
              </Link>
            ) : (
              <span>{s.label}</span>
            )}
            {s.note && !s.done ? <span className="mt-0.5 block text-[11px] font-normal text-neutral-600">{s.note}</span> : null}
          </li>
        ))}
      </ol>
      <p className="mt-3 text-[11px] text-neutral-600">
        <Link href="/help/getting-started" className="text-blue-800 underline">
          Longer in-app guide
        </Link>
        {" · "}
        <Link href="/control-plane/sources" className="text-blue-800 underline">
          Sources hub
        </Link>
      </p>
    </section>
  );
}
