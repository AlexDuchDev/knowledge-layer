import type { ReactNode } from "react";
import { PageHeader } from "@/components/shared/PageHeader";

export function SetupWizardTemplate({
  title,
  description,
  stepLabel,
  progress,
  children,
  nav,
}: {
  title: string;
  description?: ReactNode;
  stepLabel: string;
  progress?: ReactNode;
  children: ReactNode;
  nav?: ReactNode;
}) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <PageHeader title={title} description={description} screenType="Setup" />
      <div className="mb-6 rounded-lg border border-neutral-200 bg-white p-4">
        <p className="text-xs font-medium uppercase tracking-wide text-neutral-500">Step</p>
        <p className="text-sm font-medium text-neutral-900">{stepLabel}</p>
        {progress ? <div className="mt-3">{progress}</div> : null}
      </div>
      <div className="rounded-xl border border-neutral-200 bg-white p-6 shadow-sm">{children}</div>
      {nav ? <div className="mt-6 flex flex-wrap justify-between gap-2">{nav}</div> : null}
    </main>
  );
}
