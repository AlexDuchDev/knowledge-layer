import type { ReactNode } from "react";
import { PageHeader } from "@/components/shared/PageHeader";

export function DetailPageTemplate({
  title,
  description,
  metadata,
  children,
  actions,
}: {
  title: string;
  description?: ReactNode;
  metadata?: ReactNode;
  children: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <main className="mx-auto max-w-6xl px-6 py-10">
      <PageHeader title={title} description={description} screenType="Detail" actions={actions} />
      {metadata ? <div className="mb-8">{metadata}</div> : null}
      <div className="space-y-8">{children}</div>
    </main>
  );
}
