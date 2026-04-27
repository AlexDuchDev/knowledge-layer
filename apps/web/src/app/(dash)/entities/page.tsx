"use client";

import { useCallback, useState } from "react";
import Link from "next/link";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { apiBase, apiJson, principalUserId } from "@/lib/api";
import { AUTHORING_TEMPLATES } from "@/lib/authoringTemplates";

type Json = Record<string, unknown>;

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

export default function EntitiesPage() {
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [rows, setRows] = useState<Json[] | null>(null);
  const [offset, setOffset] = useState(0);
  const limit = 25;
  const [domains, setDomains] = useState<Json[] | null>(null);
  const [domainId, setDomainId] = useState("");
  const [etype, setEtype] = useState("reference_document");
  const [title, setTitle] = useState("");
  const [summary, setSummary] = useState("");
  const [body, setBody] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [fromId, setFromId] = useState("");
  const [toId, setToId] = useState("");
  const [relType, setRelType] = useState("related");

  const run = useCallback(async (fn: () => Promise<void>) => {
    setErr(null);
    setBusy(true);
    try {
      await fn();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, []);

  const load = useCallback(() => {
    run(async () => {
      const q = new URLSearchParams({ limit: String(limit), offset: String(offset) });
      if (domainId.trim()) q.set("domain_id", domainId.trim());
      setRows(await apiJson<Json[]>(`/entities?${q}`));
    });
  }, [run, offset, domainId]);

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <AppBreadcrumb items={[{ label: "Home", href: "/" }, { label: "Entities" }]} />
      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Entities</h1>
          <p className="mt-1 text-sm text-neutral-600">
            API {apiBase()} · principal {principalUserId()}
          </p>
        </div>
        <Link href="/" className="text-sm text-blue-700 underline">
          Home
        </Link>
      </div>
      {err ? <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}

      <section className="mb-8 rounded-lg border border-neutral-200 p-4">
        <h2 className="text-sm font-medium">List (paginated)</h2>
        <div className="mt-2 flex flex-wrap items-end gap-2">
          <button type="button" className="text-xs text-blue-700 underline" disabled={busy} onClick={() => run(async () => setDomains(await apiJson<Json[]>("/domains")))}>
            load domains
          </button>
          <select className="rounded border px-2 py-1 text-sm" value={domainId} onChange={(e) => setDomainId(e.target.value)}>
            <option value="">any granted</option>
            {(domains ?? []).map((d) => (
              <option key={asStr(d.id)} value={asStr(d.id)}>
                {asStr(d.name)}
              </option>
            ))}
          </select>
          <button
            type="button"
            className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={busy}
            onClick={() => load()}
          >
            GET /entities
          </button>
          <button type="button" className="rounded border px-2 py-1 text-xs disabled:opacity-50" disabled={busy || offset < limit} onClick={() => setOffset((o) => Math.max(0, o - limit))}>
            prev
          </button>
          <button type="button" className="rounded border px-2 py-1 text-xs disabled:opacity-50" disabled={busy} onClick={() => setOffset((o) => o + limit)}>
            next
          </button>
          <span className="text-xs text-neutral-500">
            offset {offset} limit {limit}
          </span>
        </div>
        {rows ? (
          <ul className="mt-3 space-y-2 text-sm">
            {rows.map((e) => (
              <li key={asStr(e.id)} className="flex flex-wrap items-baseline justify-between gap-2 rounded border border-neutral-100 px-2 py-1">
                <span>
                  <Link className="font-medium text-blue-800 underline" href={`/entities/${asStr(e.id)}`}>
                    {asStr(e.title)}
                  </Link>
                  <span className="ml-2 font-mono text-xs text-neutral-500">{asStr(e.id)}</span>
                </span>
                <span className="text-xs text-neutral-600">{asStr(e.type)}</span>
              </li>
            ))}
          </ul>
        ) : null}
      </section>

      <section className="mb-8 rounded-lg border border-neutral-200 p-4">
        <h2 className="text-sm font-medium">Create entity</h2>
        <p className="mt-1 text-xs text-neutral-600">
          Templates prefill structure; governance fields stay explicit (defaults: draft, derived). Adjust truth_mode when authoring canonical or mirrored content.
        </p>
        <div className="mt-2 flex max-w-xl flex-col gap-2">
          <button type="button" className="w-fit text-xs text-blue-700 underline" disabled={busy} onClick={() => run(async () => setDomains(await apiJson<Json[]>("/domains")))}>
            load domains
          </button>
          <label className="text-xs font-medium text-neutral-700">Template</label>
          <select
            className="rounded border px-2 py-1 text-sm"
            value={templateId}
            onChange={(e) => {
              const id = e.target.value;
              setTemplateId(id);
              const t = AUTHORING_TEMPLATES.find((x) => x.id === id);
              if (t) {
                setEtype(t.type);
                setTitle(t.title);
                setSummary(t.summary);
                setBody(t.body);
              }
            }}
          >
            <option value="">(none)</option>
            {AUTHORING_TEMPLATES.map((t) => (
              <option key={t.id} value={t.id}>
                {t.label} — {t.description}
              </option>
            ))}
          </select>
          <select className="rounded border px-2 py-1 text-sm" value={domainId} onChange={(e) => setDomainId(e.target.value)}>
            <option value="">domain</option>
            {(domains ?? []).map((d) => (
              <option key={asStr(d.id)} value={asStr(d.id)}>
                {asStr(d.name)}
              </option>
            ))}
          </select>
          <input className="rounded border px-2 py-1 text-sm" placeholder="type" value={etype} onChange={(e) => setEtype(e.target.value)} />
          <input className="rounded border px-2 py-1 text-sm" placeholder="title" value={title} onChange={(e) => setTitle(e.target.value)} />
          <input className="rounded border px-2 py-1 text-sm" placeholder="summary (optional)" value={summary} onChange={(e) => setSummary(e.target.value)} />
          <textarea className="min-h-[160px] rounded border px-2 py-1 font-mono text-xs" placeholder="body (optional)" value={body} onChange={(e) => setBody(e.target.value)} />
        </div>
        <button
          type="button"
          className="mt-2 rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy || !domainId || !title.trim()}
          onClick={() =>
            run(async () => {
              const payload: Record<string, unknown> = {
                domain_id: domainId,
                type: etype.trim(),
                title: title.trim(),
                sensitivity_level: 0,
                truth_mode: "derived",
                lifecycle_state: "draft",
              };
              if (summary.trim()) payload.summary = summary.trim();
              if (body.trim()) payload.body = body.trim();
              await apiJson("/entities", {
                method: "POST",
                body: JSON.stringify(payload),
              });
              setTitle("");
              setSummary("");
              setBody("");
              setTemplateId("");
              load();
            })
          }
        >
          POST /entities
        </button>
      </section>

      <section className="rounded-lg border border-neutral-200 p-4">
        <h2 className="text-sm font-medium">Add link</h2>
        <div className="mt-2 flex max-w-xl flex-col gap-2 sm:flex-row">
          <input className="flex-1 rounded border px-2 py-1 font-mono text-xs" placeholder="from entity id" value={fromId} onChange={(e) => setFromId(e.target.value)} />
          <input className="flex-1 rounded border px-2 py-1 font-mono text-xs" placeholder="to entity id" value={toId} onChange={(e) => setToId(e.target.value)} />
        </div>
        <input className="mt-2 max-w-xs rounded border px-2 py-1 text-sm" placeholder="relation_type" value={relType} onChange={(e) => setRelType(e.target.value)} />
        <button
          type="button"
          className="mt-2 rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy || !fromId.trim() || !toId.trim()}
          onClick={() =>
            run(async () => {
              await apiJson(`/entities/${fromId.trim()}/links`, {
                method: "POST",
                body: JSON.stringify({ to_entity_id: toId.trim(), relation_type: relType.trim() || "related" }),
              });
            })
          }
        >
          POST /entities/:id/links
        </button>
      </section>
    </main>
  );
}
