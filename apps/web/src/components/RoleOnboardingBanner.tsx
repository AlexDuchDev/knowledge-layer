"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import type { NavigationVisibility } from "@/lib/navigation";

const STORAGE_KEY = "kl_role_onboarding_dismissed";

export function RoleOnboardingBanner({ nav, pathname }: { nav: NavigationVisibility | null; pathname: string | null }) {
  const [dismissed, setDismissed] = useState(true);

  useEffect(() => {
    if (typeof window === "undefined") return;
    setDismissed(localStorage.getItem(STORAGE_KEY) === "1");
  }, []);

  const copy = useMemo(() => {
    if (!nav) return null;
    if (nav.may_manage_source_feed) {
      return {
        title: "Operator quick start",
        body: "Use the source feed wizard, run a sync, then check Governance for review queues.",
        links: [
          { href: "/control-plane/sources", label: "Sources hub" },
          { href: "/source-feeds?from=cp", label: "Connect data (wizard)" },
          { href: "/control-plane/sources/connectors", label: "Connectors" },
          { href: "/help/getting-started", label: "Guide" },
          { href: "/control-plane/jobs", label: "Knowledge jobs" },
        ],
      };
    }
    if (nav.may_approve) {
      return {
        title: "Reviewer / approver",
        body: "Open Approvals for publish gates and Notifications for open tasks.",
        links: [
          { href: "/approvals", label: "Approvals" },
          { href: "/reviews", label: "Reviews" },
          { href: "/notifications", label: "Notifications" },
        ],
      };
    }
    if (nav.may_publish) {
      return {
        title: "Author / domain steward",
        body: "Create drafts from Entities, submit for review, and track freshness in Governance.",
        links: [
          { href: "/entities", label: "Entities" },
          { href: "/governance", label: "Governance" },
          { href: "/hubs", label: "Topic hubs" },
          { href: "/help/getting-started", label: "Getting started" },
        ],
      };
    }
    if (nav.has_domain_grant) {
      return {
        title: "Knowledge user",
        body: "Search and Ask stay inside your grants; trust badges show how to read each result.",
        links: [
          { href: "/search", label: "Search" },
          { href: "/ask", label: "Ask" },
          { href: "/knowledge", label: "Browse" },
        ],
      };
    }
    return null;
  }, [nav]);

  if (dismissed || !copy || pathname?.startsWith("/login")) {
    return null;
  }

  return (
    <div className="border-b border-blue-200 bg-blue-50/90 px-4 py-3 text-sm text-neutral-900">
      <div className="mx-auto flex max-w-5xl flex-wrap items-start justify-between gap-3">
        <div>
          <div className="font-semibold">{copy.title}</div>
          <p className="mt-1 text-xs text-neutral-700">{copy.body}</p>
          <ul className="mt-2 flex flex-wrap gap-2 text-xs">
            {copy.links.map((l) => (
              <li key={l.href}>
                <Link href={l.href} className="text-blue-800 underline">
                  {l.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
        <button
          type="button"
          className="shrink-0 rounded border border-neutral-300 bg-white px-2 py-1 text-xs"
          onClick={() => {
            localStorage.setItem(STORAGE_KEY, "1");
            setDismissed(true);
          }}
        >
          Dismiss
        </button>
      </div>
    </div>
  );
}
