import type { ReactNode } from "react";
import Link from "next/link";
import { DocHelpCallout } from "@/components/guidance/DocHelpCallout";
import { PageHeader } from "@/components/shared/PageHeader";
import { SectionHeader } from "@/components/shared/SectionHeader";
import type { GuidanceSlug } from "@/lib/docConcepts";

export function AppScaffold({
  title,
  description,
  mode,
  children,
  crossLinks,
  guidanceSlug,
}: {
  title: string;
  description?: ReactNode;
  mode: "Ask" | "Search" | "Explore" | "Digest" | "Governance" | "Project" | "Decision" | "Find";
  children?: ReactNode;
  crossLinks?: { href: string; label: string }[];
  guidanceSlug?: GuidanceSlug;
}) {
  return (
    <main className="mx-auto max-w-6xl px-6 py-10">
      <PageHeader title={title} description={description} screenType={mode} />
      {guidanceSlug ? <DocHelpCallout slug={guidanceSlug} /> : null}
      {children}
      {crossLinks && crossLinks.length > 0 ? (
        <section className="mt-10">
          <SectionHeader title="Related" description="Cross-navigation and legacy mirrors" />
          <ul className="flex flex-wrap gap-2 text-sm">
            {crossLinks.map((l) => (
              <li key={l.href}>
                <Link href={l.href} className="text-blue-700 underline hover:text-blue-900">
                  {l.label}
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </main>
  );
}
