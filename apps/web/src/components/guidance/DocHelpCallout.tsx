"use client";

import Link from "next/link";
import { DOC_CONCEPTS, type GuidanceSlug } from "@/lib/docConcepts";
import { docsArticleUrl } from "@/lib/docsLinks";

export function DocHelpCallout({ slug }: { slug: GuidanceSlug }) {
  const c = DOC_CONCEPTS[slug];
  const href = docsArticleUrl(c.docFile);

  return (
    <aside className="mb-8 rounded-lg border border-blue-100 bg-blue-50/80 px-4 py-3 text-sm text-neutral-800">
      <details>
        <summary className="cursor-pointer font-medium text-blue-900">{c.title}</summary>
        <p className="mt-2 leading-relaxed text-neutral-700">{c.summary}</p>
        <p className="mt-2">
          <Link href="/help/getting-started" className="font-medium text-blue-800 underline hover:text-blue-950">
            In-app getting started
          </Link>
        </p>
        {href ? (
          <p className="mt-2">
            <a
              href={href}
              className="font-medium text-blue-800 underline hover:text-blue-950"
              target="_blank"
              rel="noreferrer"
            >
              Read full doc in repository
            </a>
          </p>
        ) : (
          <p className="mt-2 text-xs text-neutral-500">
            Set <code className="rounded bg-white px-1">NEXT_PUBLIC_DOCS_BASE_URL</code> to your GitHub{" "}
            <code className="rounded bg-white px-1">.../blob/main/docs</code> path to enable doc links.
          </p>
        )}
      </details>
    </aside>
  );
}
