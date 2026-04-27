"use client";

import Link from "next/link";

export type Crumb = { label: string; href?: string };

export function AppBreadcrumb({ items }: { items: Crumb[] }) {
  if (items.length === 0) return null;
  return (
    <nav aria-label="Breadcrumb" className="mb-4 text-sm text-neutral-600">
      <ol className="flex flex-wrap items-center gap-1">
        {items.map((c, i) => (
          <li key={`${c.label}-${i}`} className="flex items-center gap-1">
            {i > 0 ? <span className="text-neutral-400">/</span> : null}
            {c.href ? (
              <Link href={c.href} className="text-blue-700 underline hover:text-blue-900">
                {c.label}
              </Link>
            ) : (
              <span className="font-medium text-neutral-900">{c.label}</span>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
}
