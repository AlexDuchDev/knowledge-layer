import type { ReactNode } from "react";

export function PageHeader({
  title,
  description,
  screenType,
  actions,
}: {
  title: string;
  description?: ReactNode;
  /** Catalog | Builder | Preview | Operational | Binding — scaffolds surface type */
  screenType?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <h1 className="text-2xl font-semibold tracking-tight text-neutral-900">{title}</h1>
          {screenType ? (
            <span className="rounded-md bg-neutral-100 px-2 py-0.5 text-xs font-medium uppercase tracking-wide text-neutral-600">{screenType}</span>
          ) : null}
        </div>
        {description ? <div className="mt-1 max-w-3xl text-sm text-neutral-600">{description}</div> : null}
      </div>
      {actions ? <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  );
}
