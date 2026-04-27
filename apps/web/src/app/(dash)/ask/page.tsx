"use client";

import Link from "next/link";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { AskPanel } from "@/components/AskPanel";
import { PartialViewNotice } from "@/components/PartialViewNotice";

export default function AskLandingPage() {
  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Ask" }]} />
      <h1 className="text-2xl font-semibold tracking-tight">Ask</h1>
      <p className="mt-2 text-sm text-neutral-600">
        Natural-language answers are synthesized only from <strong>permission-scoped</strong> search hits, with citations and traceability. Answers are not an
        authority layer. For lookup and filters, use{" "}
        <Link className="text-blue-700 underline" href="/search">
          Search
        </Link>
        ; for deep context on one object, open an entity and use Ask there.
      </p>
      <div className="mt-4">
        <PartialViewNotice />
      </div>
      <AskPanel variant="global" />
    </main>
  );
}
