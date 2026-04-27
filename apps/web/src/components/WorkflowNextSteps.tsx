"use client";

import Link from "next/link";
import { useState } from "react";

export type WorkflowNextStepsVariant = "default" | "operator";

const DEFAULT_LINKS: { href: string; label: string }[] = [
  { href: "/search", label: "Search" },
  { href: "/ask", label: "Ask" },
  { href: "/governance", label: "Governance" },
  { href: "/entities", label: "Entities" },
];

const OPERATOR_LINKS: { href: string; label: string }[] = [
  { href: "/search", label: "Search" },
  { href: "/control-plane/sources", label: "Sources hub" },
  { href: "/source-feeds?from=cp", label: "Connect data" },
  { href: "/control-plane/governance", label: "Control plane" },
  { href: "/governance", label: "Governance hub" },
];

const MORE_LINKS: { href: string; label: string }[] = [
  { href: "/approvals", label: "Approvals" },
  { href: "/reviews", label: "Reviews" },
  { href: "/control-plane/sources/connectors", label: "Connectors" },
  { href: "/notifications", label: "Notifications" },
];

/** Compact next-step links; default shows the local golden-path slice; “More” holds secondary destinations. */
export function WorkflowNextSteps({ variant = "default" }: { variant?: WorkflowNextStepsVariant }) {
  const [moreOpen, setMoreOpen] = useState(false);
  const primary = variant === "operator" ? OPERATOR_LINKS : DEFAULT_LINKS;

  return (
    <div className="flex flex-wrap items-center gap-2 text-xs text-neutral-700">
      <span className="font-medium text-neutral-800">Next:</span>
      {primary.map((l) => (
        <Link key={l.href} href={l.href} className="text-blue-800 underline">
          {l.label}
        </Link>
      ))}
      <button
        type="button"
        className="rounded border border-neutral-200 bg-white px-1.5 py-0.5 text-[11px] text-neutral-600 hover:bg-neutral-50"
        onClick={() => setMoreOpen((o) => !o)}
        aria-expanded={moreOpen}
      >
        {moreOpen ? "Fewer" : "More"}
      </button>
      {moreOpen ? (
        <span className="flex w-full flex-wrap gap-2 border-t border-neutral-100 pt-2 sm:w-auto sm:border-0 sm:pt-0">
          {MORE_LINKS.map((l) => (
            <Link key={l.href} href={l.href} className="text-blue-800 underline">
              {l.label}
            </Link>
          ))}
        </span>
      ) : null}
    </div>
  );
}
