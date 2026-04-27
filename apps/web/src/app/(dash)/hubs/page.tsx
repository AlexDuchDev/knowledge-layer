"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { apiJson } from "@/lib/api";

type Hub = { id: string; domain_id: string; slug: string; title: string; status: string };

export default function HubsPage() {
  const [hubs, setHubs] = useState<Hub[] | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        setHubs(await apiJson<Hub[]>("/content-hubs"));
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setHubs([]);
      }
    })();
  }, []);

  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Topic hubs" }]} />
      <h1 className="text-2xl font-semibold">Topic hubs</h1>
      <p className="mt-2 text-sm text-neutral-600">Curated collections of governed entities.</p>
      {err ? <p className="mt-4 text-sm text-red-700">{err}</p> : null}
      <ul className="mt-6 space-y-2">
        {(hubs ?? []).map((h) => (
          <li key={h.id}>
            <Link href={`/hubs/${h.id}`} className="text-blue-700 underline">
              {h.title}
            </Link>
            <span className="ml-2 text-xs text-neutral-500">{h.slug}</span>
          </li>
        ))}
      </ul>
      {hubs && hubs.length === 0 && !err ? <p className="mt-4 text-sm text-neutral-600">No hubs yet. Create one via API POST /content-hubs.</p> : null}
    </main>
  );
}
