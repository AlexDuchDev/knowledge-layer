import Link from "next/link";
import type { ReactNode } from "react";

export function EmptyStateGuidance({
  title,
  body,
  primaryAction,
  secondaryActions,
}: {
  title: string;
  body: ReactNode;
  primaryAction?: { href: string; label: string };
  secondaryActions?: { href: string; label: string }[];
}) {
  return (
    <div className="mt-6 rounded-lg border border-dashed border-neutral-300 bg-neutral-50/80 px-6 py-8">
      <h2 className="text-base font-semibold text-neutral-900">{title}</h2>
      <div className="mt-2 text-sm leading-relaxed text-neutral-600">{body}</div>
      <div className="mt-4 flex flex-wrap gap-2">
        {primaryAction ? (
          <Link
            href={primaryAction.href}
            className="inline-flex rounded-md bg-blue-700 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-800"
          >
            {primaryAction.label}
          </Link>
        ) : null}
        {secondaryActions?.map((a) => (
          <Link
            key={a.href}
            href={a.href}
            className="inline-flex rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm text-neutral-800 hover:bg-neutral-50"
          >
            {a.label}
          </Link>
        ))}
      </div>
    </div>
  );
}
