"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { apiJson } from "@/lib/api";

type Feed = Record<string, unknown>;

export default function AdminSourceFeedDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const [feed, setFeed] = useState<Feed | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    void (async () => {
      try {
        setFeed(await apiJson<Feed>(`/source-feeds/${encodeURIComponent(id)}`));
        setErr(null);
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setFeed(null);
      }
    })();
  }, [id]);

  const title = typeof feed?.display_name === "string" ? feed.display_name : typeof feed?.name === "string" ? feed.name : "Source feed";

  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <AppBreadcrumb
        items={[
          { label: "Home", href: "/" },
          { label: "Source feeds", href: "/control-plane/sources" },
          { label: title },
        ]}
      />
      <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
      {err ? <div className="my-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      {feed ? <pre className="mt-4 overflow-x-auto rounded border bg-white p-4 text-xs">{JSON.stringify(feed, null, 2)}</pre> : null}
      <p className="mt-8 text-sm">
        <Link href="/control-plane/sources" className="text-blue-700 underline">
          Back to source feeds
        </Link>
      </p>
    </main>
  );
}
