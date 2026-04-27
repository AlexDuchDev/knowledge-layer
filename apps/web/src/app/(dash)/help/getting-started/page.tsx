import Link from "next/link";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";

export default function GettingStartedPage() {
  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Getting started" }]} />
      <h1 className="mt-4 text-2xl font-semibold tracking-tight text-neutral-900">Getting started</h1>
      <p className="mt-2 text-sm text-neutral-600">
        Knowledge Layer splits everyday use (Search, Ask, entities) from operator work (sources, jobs, roles). Use the header switcher{" "}
        <strong>Product</strong> / <strong>Control plane</strong> on any screen size; on phones, open <strong>Menu</strong> for the same navigation.
      </p>

      <ol className="mt-8 list-decimal space-y-6 pl-5 text-sm text-neutral-800">
        <li>
          <p className="font-medium text-neutral-900">Bootstrap the instance</p>
          <p className="mt-1 text-neutral-600">
            If no workspace exists yet, run{" "}
            <Link href="/bootstrap" className="text-blue-800 underline">
              first-time bootstrap
            </Link>{" "}
            (creates admin, domain, grants).
          </p>
        </li>
        <li>
          <p className="font-medium text-neutral-900">Connect a data source</p>
          <p className="mt-1 text-neutral-600">
            Open the{" "}
            <Link href="/control-plane/sources" className="text-blue-800 underline">
              Sources hub
            </Link>
            , pick a connector, then use the{" "}
            <Link href="/source-feeds?from=cp" className="text-blue-800 underline">
              guided source feed wizard
            </Link>{" "}
            (domain, owner, sensitivity, connector credentials). Activate when validation passes.
          </p>
        </li>
        <li>
          <p className="font-medium text-neutral-900">Run a sync</p>
          <p className="mt-1 text-neutral-600">
            After activation, trigger ingestion: open the feed&apos;s{" "}
            <Link href="/control-plane/sources" className="text-blue-800 underline">
              detail
            </Link>{" "}
            from the hub list or use the sync route under{" "}
            <code className="rounded bg-neutral-100 px-1">/source-feeds/&lt;id&gt;/sync</code>. Ensure the connector worker is running (Docker compose).
          </p>
        </li>
        <li>
          <p className="font-medium text-neutral-900">Use the product</p>
          <p className="mt-1 text-neutral-600">
            Try{" "}
            <Link href="/search" className="text-blue-800 underline">
              Search
            </Link>{" "}
            and{" "}
            <Link href="/ask" className="text-blue-800 underline">
              Ask
            </Link>{" "}
            within your domain grants. Governance queues live under{" "}
            <Link href="/governance" className="text-blue-800 underline">
              Governance
            </Link>
            .
          </p>
        </li>
      </ol>

      <p className="mt-10 text-xs text-neutral-500">
        Canonical architecture and route map: repository docs{" "}
        <code className="rounded bg-neutral-100 px-1">docs/user-facing-product-surface.md</code> and{" "}
        <code className="rounded bg-neutral-100 px-1">docs/control-plane-ui-ia.md</code>.
      </p>
    </main>
  );
}
