import type { ReactNode } from "react";
import { PageHeader } from "@/components/shared/PageHeader";

export function QueuePageTemplate({
  title,
  description,
  filters,
  actions,
  children,
}: {
  title: string;
  description?: ReactNode;
  filters?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <main className="mx-auto max-w-6xl px-6 py-10">
      <PageHeader title={title} description={description} screenType="Operational" actions={actions} />
      {filters ? <div className="mb-4">{filters}</div> : null}
      <div className="space-y-3">{children}</div>
    </main>
  );
}
