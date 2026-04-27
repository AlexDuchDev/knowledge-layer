import type { ReactNode } from "react";
import { PageHeader } from "@/components/shared/PageHeader";

export function AskSearchPageTemplate({
  title,
  description,
  mode,
  beforeQuerySlot,
  querySlot,
  filtersSlot,
  resultsSlot,
}: {
  title: string;
  description?: ReactNode;
  mode: "Ask" | "Search";
  beforeQuerySlot?: ReactNode;
  querySlot: ReactNode;
  filtersSlot?: ReactNode;
  resultsSlot: ReactNode;
}) {
  return (
    <main className="mx-auto max-w-4xl px-6 py-10">
      <PageHeader title={title} description={description} screenType={mode} />
      <div className="space-y-4">
        {beforeQuerySlot}
        {querySlot}
        {filtersSlot}
        <div className="min-h-[12rem] rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">{resultsSlot}</div>
      </div>
    </main>
  );
}
