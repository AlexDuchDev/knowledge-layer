import Link from "next/link";
import { ExploreFromHereClient } from "@/components/ExploreFromHereClient";

export default async function ExploreFromHerePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <main className="mx-auto max-w-5xl px-6 py-8">
      <div className="mb-6">
        <p className="text-xs uppercase tracking-wide text-gray-500">Bounded co-mention traversal</p>
        <h1 className="mt-1 text-2xl font-semibold text-gray-900">Explore from here</h1>
        <p className="mt-2 text-sm text-gray-600">
          One-hop neighbours of this entity, derived from the GraphRAG co-mention graph and filtered by your view permissions.
          Use this to find related decisions, policies, or projects without running a full Search query.
        </p>
        <p className="mt-2 text-xs text-gray-500">
          <Link href={`/entities/${encodeURIComponent(id)}`} className="text-blue-700 underline hover:text-blue-900">
            ← Back to entity detail
          </Link>
        </p>
      </div>
      <ExploreFromHereClient entityId={id} />
    </main>
  );
}
