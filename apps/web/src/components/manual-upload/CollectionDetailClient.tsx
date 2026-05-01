"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson, formatApiClientError } from "@/lib/api";
import { UploadModal } from "@/components/manual-upload/UploadModal";

type Collection = {
  feed_id: string;
  label: string;
  description: string;
  domain_id: string;
  sensitivity_level: number;
  status: string;
  artifact_count: number;
  last_upload_at?: string | null;
  created_at: string;
};

type Artifact = {
  id: string;
  type: string;
  title: string;
  source_ref?: string;
  warnings?: string[];
  normalized: boolean;
  created_at: string;
};

type SearchHit = {
  artifact_id: string;
  artifact_type: string;
  artifact_title: string;
  chunk_id: string;
  ordinal: number;
  snippet: string;
};

const TYPE_LABELS: Record<string, string> = {
  manual_text: "Text",
  manual_file: "File",
  manual_url: "Web page",
  manual_youtube: "YouTube",
};

export function CollectionDetailClient({ feedId }: { feedId: string }) {
  const [collection, setCollection] = useState<Collection | null>(null);
  const [artifacts, setArtifacts] = useState<Artifact[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showUpload, setShowUpload] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editLabel, setEditLabel] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [savingMeta, setSavingMeta] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchHits, setSearchHits] = useState<SearchHit[] | null>(null);
  const [searching, setSearching] = useState(false);
  const [actionBusy, setActionBusy] = useState<string | null>(null);

  async function load() {
    try {
      const [c, a] = await Promise.all([
        apiJson<Collection>(`/api/manual/collections/${encodeURIComponent(feedId)}`),
        apiJson<Artifact[] | null>(`/api/manual/collections/${encodeURIComponent(feedId)}/artifacts`),
      ]);
      setCollection(c);
      setArtifacts(a ?? []);
      setError(null);
    } catch (e) {
      setError(formatApiClientError(e));
    }
  }

  useEffect(() => {
    if (feedId) void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [feedId]);

  function startEdit() {
    if (!collection) return;
    setEditLabel(collection.label);
    setEditDescription(collection.description);
    setEditing(true);
  }

  async function saveEdit(ev: React.FormEvent) {
    ev.preventDefault();
    if (!editLabel.trim()) {
      setError("Label cannot be empty");
      return;
    }
    setSavingMeta(true);
    try {
      const updated = (await apiJson(`/api/manual/collections/${encodeURIComponent(feedId)}`, {
        method: "PATCH",
        body: JSON.stringify({ label: editLabel.trim(), description: editDescription }),
      })) as Collection;
      setCollection(updated);
      setEditing(false);
      setError(null);
    } catch (e) {
      setError(formatApiClientError(e));
    } finally {
      setSavingMeta(false);
    }
  }

  async function archive() {
    if (!confirm("Archive this collection? It will hide from lists and stop accepting uploads.")) return;
    try {
      await apiJson(`/api/manual/collections/${encodeURIComponent(feedId)}`, { method: "DELETE" });
      window.location.href = "/control-plane/sources/collections";
    } catch (e) {
      setError(formatApiClientError(e));
    }
  }

  async function deleteArtifact(id: string, title: string) {
    if (!confirm(`Delete "${title || "untitled"}" and its chunks/embeddings? This cannot be undone.`)) return;
    setActionBusy(id);
    try {
      await apiJson(`/api/manual/collections/${encodeURIComponent(feedId)}/artifacts/${encodeURIComponent(id)}`, {
        method: "DELETE",
      });
      await load();
      // Drop matching search hits if a search is currently displayed.
      if (searchHits) {
        setSearchHits(searchHits.filter((h) => h.artifact_id !== id));
      }
    } catch (e) {
      setError(formatApiClientError(e));
    } finally {
      setActionBusy(null);
    }
  }

  async function renormalize(id: string) {
    setActionBusy(id);
    try {
      await apiJson(`/api/manual/collections/${encodeURIComponent(feedId)}/artifacts/${encodeURIComponent(id)}/renormalize`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      await load();
    } catch (e) {
      setError(formatApiClientError(e));
    } finally {
      setActionBusy(null);
    }
  }

  async function runSearch(ev: React.FormEvent) {
    ev.preventDefault();
    const q = searchQuery.trim();
    if (!q) {
      setSearchHits(null);
      return;
    }
    setSearching(true);
    try {
      const res = (await apiJson(`/api/manual/collections/${encodeURIComponent(feedId)}/search`, {
        method: "POST",
        body: JSON.stringify({ q, limit: 25 }),
      })) as { hits: SearchHit[] };
      setSearchHits(res.hits ?? []);
      setError(null);
    } catch (e) {
      setError(formatApiClientError(e));
    } finally {
      setSearching(false);
    }
  }

  if (error && !collection) {
    return <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{error}</div>;
  }
  if (!collection) {
    return <p className="text-sm text-gray-500">Loading…</p>;
  }

  return (
    <div className="space-y-6">
      <div className="rounded-md border bg-white p-4">
        {!editing ? (
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <h2 className="text-lg font-semibold">{collection.label || "Untitled"}</h2>
              {collection.description ? (
                <p className="mt-1 text-sm text-gray-600">{collection.description}</p>
              ) : null}
              <p className="mt-2 text-xs text-gray-500">
                Sensitivity {collection.sensitivity_level} · status {collection.status} · {collection.artifact_count}{" "}
                artifact(s)
              </p>
            </div>
            <div className="flex flex-col gap-2">
              <button
                type="button"
                onClick={() => setShowUpload(true)}
                className="rounded-md bg-blue-700 px-3 py-1.5 text-sm text-white hover:bg-blue-800"
              >
                + Add content
              </button>
              <button
                type="button"
                onClick={startEdit}
                className="rounded-md border border-gray-300 px-3 py-1.5 text-xs text-gray-800 hover:bg-gray-50"
              >
                Edit
              </button>
              <button
                type="button"
                onClick={() => void archive()}
                className="rounded-md border border-red-300 px-3 py-1.5 text-xs text-red-700 hover:bg-red-50"
              >
                Archive
              </button>
            </div>
          </div>
        ) : (
          <form onSubmit={saveEdit} className="space-y-3">
            <div>
              <label className="block text-xs font-medium text-gray-700">Label</label>
              <input
                type="text"
                value={editLabel}
                onChange={(e) => setEditLabel(e.target.value)}
                className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
                required
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700">Description</label>
              <input
                type="text"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
              />
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                disabled={savingMeta}
                className="rounded-md bg-blue-700 px-3 py-1.5 text-sm text-white hover:bg-blue-800 disabled:bg-gray-400"
              >
                {savingMeta ? "Saving…" : "Save"}
              </button>
              <button
                type="button"
                onClick={() => setEditing(false)}
                className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-800 hover:bg-gray-50"
              >
                Cancel
              </button>
            </div>
          </form>
        )}
      </div>

      <section>
        <form onSubmit={runSearch} className="flex gap-2">
          <input
            type="search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search this collection…"
            className="flex-1 rounded-md border border-gray-300 px-3 py-1.5 text-sm"
          />
          <button
            type="submit"
            disabled={searching}
            className="rounded-md bg-gray-900 px-3 py-1.5 text-sm text-white hover:bg-black disabled:bg-gray-400"
          >
            {searching ? "Searching…" : "Search"}
          </button>
          {searchHits ? (
            <button
              type="button"
              onClick={() => {
                setSearchHits(null);
                setSearchQuery("");
              }}
              className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-800 hover:bg-gray-50"
            >
              Clear
            </button>
          ) : null}
        </form>
        {searchHits ? (
          <div className="mt-3">
            {searchHits.length === 0 ? (
              <p className="text-sm text-gray-500">No matches in this collection.</p>
            ) : (
              <ul className="space-y-2">
                {searchHits.map((h) => (
                  <li key={h.chunk_id} className="rounded-md border bg-white p-3 text-sm">
                    <div className="flex items-center justify-between gap-2 text-xs text-gray-500">
                      <span>
                        {TYPE_LABELS[h.artifact_type] ?? h.artifact_type} · chunk #{h.ordinal}
                      </span>
                    </div>
                    <p className="mt-1 font-medium text-gray-900">{h.artifact_title || "Untitled"}</p>
                    <p className="mt-1 whitespace-pre-line text-gray-700">{h.snippet}</p>
                  </li>
                ))}
              </ul>
            )}
          </div>
        ) : null}
      </section>

      <section>
        <h3 className="mb-2 text-sm font-semibold uppercase tracking-wide text-gray-600">Artifacts</h3>
        {error && collection ? (
          <div className="mb-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-900">{error}</div>
        ) : null}
        {artifacts === null ? (
          <p className="text-sm text-gray-500">Loading…</p>
        ) : artifacts.length === 0 ? (
          <p className="text-sm text-gray-500">Nothing here yet. Add files, paste text, or import a URL/video.</p>
        ) : (
          <div className="overflow-x-auto rounded-md border bg-white">
            <table className="min-w-full text-sm">
              <thead className="bg-gray-50 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                <tr>
                  <th className="px-3 py-2">Title</th>
                  <th className="px-3 py-2">Type</th>
                  <th className="px-3 py-2">Status</th>
                  <th className="px-3 py-2">Added</th>
                  <th className="px-3 py-2 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {artifacts.map((a) => (
                  <tr key={a.id}>
                    <td className="px-3 py-2">
                      <div className="font-medium text-gray-900">{a.title || "Untitled"}</div>
                      {a.source_ref ? <div className="text-xs text-gray-500">{a.source_ref}</div> : null}
                      {a.warnings && a.warnings.length > 0 ? (
                        <ul className="mt-1 list-disc pl-4 text-xs text-amber-700">
                          {a.warnings.map((w, i) => (
                            <li key={i}>{w}</li>
                          ))}
                        </ul>
                      ) : null}
                    </td>
                    <td className="px-3 py-2 text-xs text-gray-700">{TYPE_LABELS[a.type] ?? a.type}</td>
                    <td className="px-3 py-2 text-xs">
                      {a.normalized ? (
                        <span className="rounded bg-green-100 px-2 py-0.5 text-green-800">indexed</span>
                      ) : (
                        <span className="rounded bg-amber-100 px-2 py-0.5 text-amber-800">pending</span>
                      )}
                    </td>
                    <td className="px-3 py-2 text-xs text-gray-600">{new Date(a.created_at).toLocaleString()}</td>
                    <td className="px-3 py-2 text-right">
                      <div className="inline-flex gap-1">
                        <button
                          type="button"
                          onClick={() => void renormalize(a.id)}
                          disabled={actionBusy === a.id}
                          title="Re-extract and re-index"
                          className="rounded border border-gray-300 px-2 py-0.5 text-xs text-gray-700 hover:bg-gray-50 disabled:bg-gray-100"
                        >
                          Re-index
                        </button>
                        <button
                          type="button"
                          onClick={() => void deleteArtifact(a.id, a.title)}
                          disabled={actionBusy === a.id}
                          className="rounded border border-red-300 px-2 py-0.5 text-xs text-red-700 hover:bg-red-50 disabled:bg-red-50"
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <p className="text-sm">
        <Link href="/control-plane/sources/collections" className="text-blue-700 underline hover:text-blue-900">
          ← Back to collections
        </Link>
      </p>

      {showUpload ? (
        <UploadModal
          feedId={feedId}
          onClose={() => setShowUpload(false)}
          onUploaded={() => {
            void load();
          }}
        />
      ) : null}
    </div>
  );
}
