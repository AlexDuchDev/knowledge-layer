"use client";

/**
 * Explore from here (Phase 2.3.1).
 *
 * Calls GET /entities/:id/graph-explore?max_nodes=… and renders the
 * permission-filtered list of co-mention neighbours. Surfaces denied_count
 * so operators can tell when the graph was trimmed by ACLs vs sparse.
 *
 * Requires NEO4J_URL on the server; without it the API returns 503 and the
 * UI shows a clear message linking to the optional-modules section in OSS_V1_SCOPE.
 */

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson, formatApiClientError } from "@/lib/api";

type EntitySummary = {
  id: string;
  type: string;
  title: string;
  domain_id: string;
  sensitivity_level: number;
  truth_mode: string;
  lifecycle_state: string;
  freshness_status: string;
  approval_status: string;
};

type Neighbour = {
  entity: EntitySummary;
  mention_count: number;
};

type GraphExploreResponse = {
  seed_entity_id: string;
  neighbours: Neighbour[];
  returned: number;
  denied_count: number;
  truncated: boolean;
};

const MAX_NODES_OPTIONS = [12, 24, 48, 100];

export function ExploreFromHereClient({ entityId }: { entityId: string }) {
  const [maxNodes, setMaxNodes] = useState(24);
  const [data, setData] = useState<GraphExploreResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [unavailable, setUnavailable] = useState(false);

  async function load() {
    setLoading(true);
    setError(null);
    setUnavailable(false);
    try {
      const res = await apiJson<GraphExploreResponse>(`/entities/${encodeURIComponent(entityId)}/graph-explore?max_nodes=${maxNodes}`);
      setData(res);
    } catch (err) {
      const msg = formatApiClientError(err);
      if (msg.includes("503") || msg.toLowerCase().includes("graphrag")) {
        setUnavailable(true);
      } else {
        setError(msg);
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [maxNodes, entityId]);

  if (unavailable) {
    return (
      <div className="rounded-md border border-yellow-200 bg-yellow-50 p-4 text-sm text-yellow-900">
        <p className="font-semibold">GraphRAG not configured.</p>
        <p className="mt-1">
          &ldquo;Explore from here&rdquo; uses the optional GraphRAG module (Neo4j). Set <code className="font-mono">NEO4J_URL</code> on the API and worker
          processes, then re-ingest entities to populate the graph. See the &ldquo;Optional modules&rdquo; section of OSS_V1_SCOPE.md.
        </p>
        <p className="mt-2">
          <Link href={`/entities/${encodeURIComponent(entityId)}`} className="text-blue-700 underline hover:text-blue-900">
            Back to entity
          </Link>
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end gap-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <label className="flex flex-col text-xs font-medium text-gray-700">
          <span>Max neighbours</span>
          <select value={maxNodes} onChange={(e) => setMaxNodes(Number(e.target.value))} className="mt-1 rounded-md border border-gray-300 px-2 py-1 text-sm">
            {MAX_NODES_OPTIONS.map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </label>
        <button onClick={load} disabled={loading} className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400">
          {loading ? "Loading…" : "Refresh"}
        </button>
        {data ? (
          <div className="flex flex-wrap gap-x-4 text-xs text-gray-600">
            <span>{data.returned} visible</span>
            {data.denied_count > 0 ? (
              <span className="text-amber-700">+{data.denied_count} hidden by access policy</span>
            ) : null}
            {data.truncated ? <span>(limit reached)</span> : null}
          </div>
        ) : null}
      </div>

      {error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</div>
      ) : null}

      {data ? (
        data.neighbours.length === 0 ? (
          <div className="rounded-md border border-gray-200 bg-white p-6 text-sm text-gray-600">
            No neighbours visible to you. {data.denied_count > 0 ? `Permission policy hid ${data.denied_count} neighbour(s).` : "The graph may still be building, or this entity has no co-mentions yet."}
          </div>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white shadow-sm">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50 text-xs uppercase tracking-wide text-gray-600">
                <tr>
                  <Th>Title</Th>
                  <Th>Type</Th>
                  <Th>Lifecycle</Th>
                  <Th>Truth</Th>
                  <Th>Mentions</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {data.neighbours.map((n) => (
                  <tr key={n.entity.id} className="text-gray-800">
                    <Td>
                      <Link href={`/entities/${encodeURIComponent(n.entity.id)}`} className="text-blue-700 underline hover:text-blue-900">
                        {n.entity.title || n.entity.id.slice(0, 8)}
                      </Link>
                    </Td>
                    <Td><span className="font-mono text-xs">{n.entity.type}</span></Td>
                    <Td>{n.entity.lifecycle_state}</Td>
                    <Td>{n.entity.truth_mode}</Td>
                    <Td>{n.mention_count}</Td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      ) : null}
    </div>
  );
}

function Th({ children }: { children: React.ReactNode }) {
  return <th className="px-4 py-2 text-left">{children}</th>;
}
function Td({ children }: { children: React.ReactNode }) {
  return <td className="px-4 py-2">{children}</td>;
}
