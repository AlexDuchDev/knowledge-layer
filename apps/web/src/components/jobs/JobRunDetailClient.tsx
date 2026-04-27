"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson } from "@/lib/api";

type Run = Record<string, unknown>;

export function JobRunDetailClient({
  runId,
  footerBackHref,
  footerBackLabel,
}: {
  runId: string;
  footerBackHref: string;
  footerBackLabel: string;
}) {
  const [run, setRun] = useState<Run | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!runId) return;
    void (async () => {
      try {
        setRun(await apiJson<Run>(`/job-runs/${encodeURIComponent(runId)}`));
        setErr(null);
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setRun(null);
      }
    })();
  }, [runId]);

  return (
    <>
      <p className="mt-1 font-mono text-xs text-neutral-600">{runId}</p>
      {err ? <div className="my-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      {run ? <pre className="mt-4 overflow-x-auto rounded border bg-white p-4 text-xs">{JSON.stringify(run, null, 2)}</pre> : null}
      <p className="mt-8 text-sm">
        <Link href={footerBackHref} className="text-blue-700 underline">
          {footerBackLabel}
        </Link>
      </p>
    </>
  );
}
