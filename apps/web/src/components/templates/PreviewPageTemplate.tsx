import type { ReactNode } from "react";
import { PageHeader } from "@/components/shared/PageHeader";

export function PreviewPageTemplate({
  title,
  description,
  children,
  actions,
}: {
  title: string;
  description?: ReactNode;
  children: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <main className="mx-auto max-w-6xl px-6 py-10">
      <PageHeader title={title} description={description} screenType="Preview" actions={actions} />
      <div className="rounded-xl border border-dashed border-neutral-300 bg-neutral-50/50 p-6">{children}</div>
    </main>
  );
}
