import type { ReactNode } from "react";
import { PageHeader } from "@/components/shared/PageHeader";

export function BuilderPageTemplate({
  title,
  description,
  sidebar,
  children,
  footerActions,
}: {
  title: string;
  description?: ReactNode;
  sidebar?: ReactNode;
  children: ReactNode;
  footerActions?: ReactNode;
}) {
  return (
    <main className="mx-auto max-w-6xl px-6 py-10">
      <PageHeader title={title} description={description} screenType="Builder" />
      <div className={`grid gap-8 ${sidebar ? "lg:grid-cols-[1fr_20rem]" : ""}`}>
        <div className="min-w-0 space-y-6">{children}</div>
        {sidebar ? <aside className="min-w-0 space-y-4 lg:border-l lg:border-neutral-200 lg:pl-8">{sidebar}</aside> : null}
      </div>
      {footerActions ? <div className="mt-8 flex flex-wrap gap-2 border-t border-neutral-200 pt-6">{footerActions}</div> : null}
    </main>
  );
}
