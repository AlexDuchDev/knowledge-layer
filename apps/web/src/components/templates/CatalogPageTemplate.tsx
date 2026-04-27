import type { ReactNode } from "react";
import { PageHeader } from "@/components/shared/PageHeader";

export function CatalogPageTemplate({
  title,
  description,
  filters,
  children,
  actions,
}: {
  title: string;
  description?: ReactNode;
  filters?: ReactNode;
  children: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <main className="mx-auto max-w-6xl px-6 py-10">
      <PageHeader title={title} description={description} screenType="Catalog" actions={actions} />
      {filters}
      <div className="mt-2">{children}</div>
    </main>
  );
}
