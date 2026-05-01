"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiJson, formatApiClientError } from "@/lib/api";

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

type Domain = { id: string; name: string };

export function CollectionsClient() {
  const [collections, setCollections] = useState<Collection[] | null>(null);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [creating, setCreating] = useState(false);
  const [label, setLabel] = useState("");
  const [description, setDescription] = useState("");
  const [domainId, setDomainId] = useState("");
  const [sensitivity, setSensitivity] = useState(1);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function load() {
    try {
      const [list, doms] = await Promise.all([
        apiJson<Collection[] | null>("/api/manual/collections"),
        apiJson<Domain[]>("/domains").catch(() => [] as Domain[]),
      ]);
      setCollections(list ?? []);
      setDomains(doms);
      if (!domainId && doms[0]) {
        setDomainId(doms[0].id);
      }
      setError(null);
    } catch (e) {
      setError(formatApiClientError(e));
      setCollections([]);
    }
  }

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleCreate(ev: React.FormEvent) {
    ev.preventDefault();
    if (!label.trim()) {
      setError("Label is required");
      return;
    }
    if (!domainId) {
      setError("Pick a domain");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await apiJson("/api/manual/collections", {
        method: "POST",
        body: JSON.stringify({
          label: label.trim(),
          description: description.trim(),
          domain_id: domainId,
          sensitivity_level: sensitivity,
        }),
      });
      setLabel("");
      setDescription("");
      setCreating(false);
      await load();
    } catch (e) {
      setError(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600">
          Collections are folders for ad-hoc uploads — files, pasted text, web pages, and YouTube transcripts. Each collection
          is governed (domain + sensitivity) and can be the source scope for knowledge jobs.
        </p>
        {!creating ? (
          <button
            type="button"
            className="rounded-md bg-blue-700 px-3 py-1.5 text-sm text-white hover:bg-blue-800"
            onClick={() => setCreating(true)}
          >
            + New collection
          </button>
        ) : null}
      </div>

      {creating ? (
        <form onSubmit={handleCreate} className="space-y-3 rounded-md border bg-white p-4">
          <div>
            <label className="block text-xs font-medium text-gray-700">Label</label>
            <input
              type="text"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="e.g. Competitor research Q2"
              className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
              required
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Description (optional)</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What's in this collection"
              className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
            />
          </div>
          <div className="flex gap-3">
            <div className="flex-1">
              <label className="block text-xs font-medium text-gray-700">Domain</label>
              <select
                value={domainId}
                onChange={(e) => setDomainId(e.target.value)}
                className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
                required
              >
                <option value="">— Select —</option>
                {domains.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="w-44">
              <label className="block text-xs font-medium text-gray-700">Sensitivity</label>
              <select
                value={sensitivity}
                onChange={(e) => setSensitivity(Number(e.target.value))}
                className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
              >
                <option value={0}>0 — Internal open</option>
                <option value={1}>1 — Team-wide</option>
                <option value={2}>2 — Domain restricted</option>
                <option value={3}>3 — Confidential</option>
              </select>
            </div>
          </div>
          {error ? <p className="text-sm text-red-700">{error}</p> : null}
          <div className="flex gap-2">
            <button
              type="submit"
              disabled={busy}
              className="rounded-md bg-blue-700 px-3 py-1.5 text-sm text-white hover:bg-blue-800 disabled:bg-gray-400"
            >
              {busy ? "Creating…" : "Create collection"}
            </button>
            <button
              type="button"
              onClick={() => {
                setCreating(false);
                setError(null);
              }}
              className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-800 hover:bg-gray-50"
            >
              Cancel
            </button>
          </div>
        </form>
      ) : null}

      {!creating && error ? (
        <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{error}</div>
      ) : null}

      {collections === null ? (
        <p className="text-sm text-gray-500">Loading…</p>
      ) : collections.length === 0 ? (
        <p className="text-sm text-gray-500">No collections yet. Create one to start uploading.</p>
      ) : (
        <ul className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {collections.map((c) => (
            <li key={c.feed_id} className="rounded-md border bg-white p-4 hover:border-blue-400">
              <Link href={`/control-plane/sources/collections/${c.feed_id}`} className="block">
                <h3 className="text-base font-semibold text-gray-900">{c.label || "Untitled"}</h3>
                {c.description ? <p className="mt-1 text-sm text-gray-600">{c.description}</p> : null}
                <p className="mt-2 text-xs text-gray-500">
                  {c.artifact_count} {c.artifact_count === 1 ? "artifact" : "artifacts"}
                  {c.last_upload_at ? ` · last upload ${new Date(c.last_upload_at).toLocaleString()}` : null}
                </p>
                <p className="mt-1 text-xs text-gray-400">Sensitivity {c.sensitivity_level} · {c.status}</p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
