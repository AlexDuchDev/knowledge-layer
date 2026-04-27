"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { apiJson } from "@/lib/api";
import { KnowledgeCard } from "@/components/KnowledgeCard";
import { TrustExplanationDrawer } from "@/components/TrustExplanationDrawer";
import { DocHelpCallout } from "@/components/guidance/DocHelpCallout";
import { WorkflowNextSteps } from "@/components/WorkflowNextSteps";

export type AskCitation = { entity_id: string; quote?: string };
export type AskSupportingEntity = {
  entity_id: string;
  title: string;
  domain_id: string;
  entity_type: string;
  truth_mode: string;
  lifecycle_state: string;
  freshness_status: string;
};
export type AskEntityResponse = {
  trace_id: string;
  answer: string;
  citations: AskCitation[];
  supporting_entities: AskSupportingEntity[];
  scope?: Record<string, unknown>;
};

const FEEDBACK_KINDS = [
  { kind: "useful", label: "Useful" },
  { kind: "not_useful", label: "Not useful" },
  { kind: "likely_stale", label: "Likely stale" },
  { kind: "weak_citations", label: "Weak citations" },
  { kind: "possibly_incorrect", label: "Possibly incorrect" },
  { kind: "incomplete", label: "Incomplete" },
] as const;

type AskPanelProps =
  | { variant?: "entity"; entityId: string }
  | { variant: "global"; entityId?: never };

export function AskPanel(props: AskPanelProps) {
  const variant = props.variant ?? "entity";
  const entityId = variant === "entity" ? props.entityId : undefined;

  const [question, setQuestion] = useState("");
  const [includeRelated, setIncludeRelated] = useState(true);
  const [answerStrategy, setAnswerStrategy] = useState<"standard" | "best_trusted">("standard");
  const [domainId, setDomainId] = useState("");
  const [typeFilter, setTypeFilter] = useState("");
  const [domainOptions, setDomainOptions] = useState<{ id: string; name: string }[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [res, setRes] = useState<AskEntityResponse | null>(null);
  const [feedbackSent, setFeedbackSent] = useState<string | null>(null);
  const [imageFiles, setImageFiles] = useState<File[]>([]);
  const [audioFile, setAudioFile] = useState<File | null>(null);

  useEffect(() => {
    if (variant !== "global") return;
    void (async () => {
      try {
        const d = await apiJson<{ id: string; name: string }[]>("/domains");
        setDomainOptions(d);
      } catch {
        setDomainOptions([]);
      }
    })();
  }, [variant]);

  const submit = useCallback(async () => {
    setErr(null);
    setFeedbackSent(null);
    setBusy(true);
    try {
      const images =
        imageFiles.length > 0
          ? await Promise.all(
              imageFiles.slice(0, 8).map(async (f) => {
                const { data_base64, media_type } = await readFileBase64Payload(f);
                return { data_base64, media_type };
              }),
            )
          : undefined;
      let audio_base64: string | undefined;
      let audio_format: string | undefined;
      if (audioFile) {
        const { data_base64 } = await readFileBase64Payload(audioFile);
        audio_base64 = data_base64;
        audio_format = audioFormatFromFileName(audioFile.name);
      }
      const multimodal: Record<string, unknown> = {};
      if (images?.length) multimodal.images = images;
      if (audio_base64) {
        multimodal.audio_base64 = audio_base64;
        multimodal.audio_format = audio_format;
      }
      if (variant === "global") {
        const out = await apiJson<AskEntityResponse>("/ask", {
          method: "POST",
          body: JSON.stringify({
            question,
            include_related: includeRelated,
            answer_strategy: answerStrategy,
            domain_id: domainId.trim() || undefined,
            type: typeFilter.trim() || undefined,
            ...multimodal,
          }),
        });
        setRes(out);
      } else {
        const out = await apiJson<AskEntityResponse>(`/entities/${entityId}/ask`, {
          method: "POST",
          body: JSON.stringify({
            question,
            include_related: includeRelated,
            answer_strategy: answerStrategy,
            ...multimodal,
          }),
        });
        setRes(out);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setRes(null);
    } finally {
      setBusy(false);
    }
  }, [variant, entityId, question, includeRelated, answerStrategy, domainId, typeFilter, imageFiles, audioFile]);

  const feedback = useCallback(
    async (kind: string) => {
      if (!res?.trace_id) return;
      setErr(null);
      try {
        await apiJson("/answer-feedback", {
          method: "POST",
          body: JSON.stringify({
            trace_id: res.trace_id,
            feedback_kind: kind,
          }),
        });
        setFeedbackSent(kind);
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
      }
    },
    [res?.trace_id],
  );

  const citedSet = useMemo(() => {
    const s = new Set<string>();
    for (const c of res?.citations ?? []) s.add(c.entity_id);
    return s;
  }, [res?.citations]);

  const primarySupporting = useMemo(() => (res?.supporting_entities ?? []).filter((e) => citedSet.has(e.entity_id)), [res?.supporting_entities, citedSet]);
  const secondarySupporting = useMemo(
    () => (res?.supporting_entities ?? []).filter((e) => !citedSet.has(e.entity_id)),
    [res?.supporting_entities, citedSet],
  );

  const trustNotes = useMemo(() => {
    const notes: string[] = [];
    const mode = (res?.scope?.ask_mode as string) || "";
    if (mode === "global") {
      notes.push("This answer is grounded in entities returned by a permission-scoped keyword search on your question—not the whole organization.");
    }
    if (answerStrategy === "best_trusted") {
      notes.push("Best-trusted mode prefers stronger canonical / lifecycle evidence when the model can infer it from the evidence blocks.");
    }
    if (res && res.citations.length === 0 && res.answer.trim().length > 0) {
      notes.push("No citations were returned; treat the answer as provisional and open supporting entities below.");
    }
    const derivedHeavy =
      res &&
      res.supporting_entities.length > 0 &&
      res.supporting_entities.filter((e) => String(e.truth_mode).toLowerCase().includes("derived")).length > res.supporting_entities.length / 2;
    if (derivedHeavy) {
      notes.push("Much of the evidence is derived-class; prefer canonical or approved sources when available.");
    }
    const staleHeavy =
      res &&
      res.supporting_entities.length > 0 &&
      res.supporting_entities.filter((e) => String(e.freshness_status).toLowerCase() === "stale").length > res.supporting_entities.length / 2;
    if (staleHeavy) {
      notes.push("Stale freshness appears on several evidence items; verify currency before acting.");
    }
    return notes;
  }, [res, answerStrategy]);

  const citationMeta = useMemo(() => {
    const m = new Map<string, AskSupportingEntity>();
    for (const e of res?.supporting_entities ?? []) m.set(e.entity_id, e);
    return m;
  }, [res?.supporting_entities]);

  return (
    <section className="mt-8 rounded-lg border border-neutral-200 p-4">
      {variant === "global" ? <DocHelpCallout slug="ask" /> : null}
      <h2 className="text-lg font-medium">{variant === "global" ? "Ask (governed synthesis)" : "Ask about this entity"}</h2>
      <p className="mt-1 text-sm text-neutral-600">
        {variant === "global" ? (
          <>
            Questions run against <strong>permitted domains only</strong>. The API discovers candidate entities with the same search scope as{" "}
            <Link className="text-blue-700 underline" href="/search">
              Search
            </Link>
            , then synthesizes an answer with citations. This is not a global chatbot over all company data.
          </>
        ) : (
          <>
            This question is scoped to this entity{includeRelated ? " and directly linked entities (1-hop)" : ""}. No hidden access broadening.
          </>
        )}
      </p>
      <div className="mt-3 rounded-md border border-neutral-100 bg-neutral-50/80 px-2 py-2">
        <WorkflowNextSteps />
      </div>

      {err ? (
        <div className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-900">{err}</div>
      ) : null}

      <div className="mt-4 grid gap-3">
        {variant === "global" ? (
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <div className="mb-1 text-xs font-medium text-neutral-700">Optional domain filter</div>
              <select className="w-full rounded border border-neutral-300 px-3 py-2 text-sm" value={domainId} onChange={(e) => setDomainId(e.target.value)}>
                <option value="">All granted domains</option>
                {(domainOptions ?? []).map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <div className="mb-1 text-xs font-medium text-neutral-700">Optional entity type</div>
              <input
                className="w-full rounded border border-neutral-300 px-3 py-2 text-sm"
                placeholder="e.g. decision, policy"
                value={typeFilter}
                onChange={(e) => setTypeFilter(e.target.value)}
              />
            </div>
          </div>
        ) : null}
        <textarea
          className="min-h-24 w-full rounded border border-neutral-300 px-3 py-2 text-sm"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder={
            variant === "global"
              ? "What did we decide about onboarding ownership?"
              : "Ask a question about this entity…"
          }
        />
        <div className="grid gap-2 text-xs text-neutral-700">
          <label className="block">
            <span className="font-medium">Images (optional)</span>
            <input
              type="file"
              accept="image/*"
              multiple
              className="mt-1 block w-full text-xs"
              onChange={(e) => setImageFiles(Array.from(e.target.files ?? []).slice(0, 8))}
            />
            <span className="text-neutral-500">Up to 8 images; sent to vision-capable models with your question and evidence.</span>
          </label>
          <label className="block">
            <span className="font-medium">Voice note (optional)</span>
            <input type="file" accept="audio/*" className="mt-1 block w-full text-xs" onChange={(e) => setAudioFile(e.target.files?.[0] ?? null)} />
            <span className="text-neutral-500">Short clip transcribed on the server, then merged into your question.</span>
          </label>
        </div>
        <label className="flex items-center gap-2 text-sm text-neutral-800">
          <input type="checkbox" checked={includeRelated} onChange={(e) => setIncludeRelated(e.target.checked)} />
          {variant === "global"
            ? "Include 1-hop linked entities from the top search hit (same cap as entity Ask)"
            : "Include directly related evidence (1-hop)"}
        </label>
        <div>
          <div className="mb-1 text-xs font-medium text-neutral-700">Answer strategy</div>
          <select
            className="rounded border border-neutral-300 px-2 py-1 text-sm"
            value={answerStrategy}
            onChange={(e) => setAnswerStrategy(e.target.value as "standard" | "best_trusted")}
          >
            <option value="standard">Standard</option>
            <option value="best_trusted">Best trusted (prefer canonical / approved ordering)</option>
          </select>
        </div>
        <button
          type="button"
          className="w-fit rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={
            busy ||
            (question.trim() === "" && imageFiles.length === 0 && !audioFile) ||
            (variant === "entity" && (entityId ?? "").trim() === "")
          }
          onClick={submit}
        >
          {busy ? "Gathering permitted evidence…" : "Ask"}
        </button>
        {busy ? <p className="text-xs text-neutral-600">Building an answer from the strongest permitted evidence available…</p> : null}
      </div>

      {res ? (
        <div className="mt-6">
          <div className="text-xs text-neutral-600">
            Trace: <code className="rounded bg-neutral-100 px-1">{res.trace_id}</code>
          </div>
          {trustNotes.length > 0 ? (
            <ul className="mt-3 list-disc space-y-1 pl-5 text-xs text-neutral-700">
              {trustNotes.map((n) => (
                <li key={n}>{n}</li>
              ))}
            </ul>
          ) : null}
          <div className="mt-3">
            <KnowledgeCard
              variant="answer"
              density="comfortable"
              title="Answer"
              snippet={res.answer}
              citations={res.citations.map((c) => ({ entity_id: c.entity_id, quote: c.quote }))}
              footer={
                <span className="text-[10px] text-neutral-500">
                  Strategy: {answerStrategy}. Evidence ordered by trust heuristics on the API.
                </span>
              }
            />
          </div>

          <div className="mt-4 flex flex-wrap gap-2">
            {FEEDBACK_KINDS.map(({ kind, label }) => (
              <button
                key={kind}
                type="button"
                className={`rounded px-2 py-1 text-xs text-white ${feedbackSent === kind ? "bg-green-800" : "bg-neutral-900"}`}
                onClick={() => feedback(kind)}
              >
                {label}
                {feedbackSent === kind ? " ✓" : ""}
              </button>
            ))}
          </div>

          <div className="mt-6">
            <h3 className="text-sm font-medium">Citations</h3>
            {res.citations.length === 0 ? (
              <p className="mt-2 text-sm text-neutral-600">No citations returned.</p>
            ) : (
              <ul className="mt-2 space-y-2 text-sm">
                {res.citations.map((c, i) => {
                  const meta = citationMeta.get(c.entity_id);
                  return (
                    <li key={`${c.entity_id}-${i}`} className="rounded border border-neutral-200 p-2">
                      <div className="flex flex-wrap items-baseline justify-between gap-2">
                        <div>
                          <Link href={`/entities/${c.entity_id}`} className="font-medium text-blue-700 underline">
                            {meta?.title ?? c.entity_id}
                          </Link>
                          {meta ? (
                            <span className="ml-2 text-xs text-neutral-500">
                              {meta.entity_type} · {meta.truth_mode} · {meta.lifecycle_state}
                            </span>
                          ) : null}
                        </div>
                      </div>
                      {c.quote ? <p className="mt-1 text-neutral-700">“{c.quote}”</p> : null}
                    </li>
                  );
                })}
              </ul>
            )}
          </div>

          <div className="mt-6">
            <h3 className="text-sm font-medium">Supporting entities</h3>
            <p className="mt-1 text-xs text-neutral-600">Primary: cited in the answer. Secondary: useful governed objects to inspect next.</p>
            <h4 className="mt-3 text-xs font-semibold text-neutral-800">Primary</h4>
            <SupportingTable rows={primarySupporting} citedSet={citedSet} />
            <h4 className="mt-4 text-xs font-semibold text-neutral-800">Secondary</h4>
            <SupportingTable rows={secondarySupporting} citedSet={citedSet} emptyHint="No additional context entities." />
          </div>
        </div>
      ) : null}
    </section>
  );
}

async function readFileBase64Payload(f: File): Promise<{ data_base64: string; media_type: string }> {
  const buf = await f.arrayBuffer();
  const bytes = new Uint8Array(buf);
  let binary = "";
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  const data_base64 = btoa(binary);
  return { data_base64, media_type: f.type || "application/octet-stream" };
}

function audioFormatFromFileName(name: string): string {
  const m = name.toLowerCase().match(/\.([a-z0-9]+)$/);
  if (!m) return "wav";
  return m[1];
}

function SupportingTable({
  rows,
  citedSet,
  emptyHint,
}: {
  rows: AskSupportingEntity[];
  citedSet: Set<string>;
  emptyHint?: string;
}) {
  if (rows.length === 0) {
    return <p className="mt-1 text-xs text-neutral-500">{emptyHint ?? "None."}</p>;
  }
  return (
    <div className="mt-2 overflow-x-auto rounded border border-neutral-200">
      <table className="min-w-full text-left text-xs">
        <thead className="bg-neutral-50">
          <tr>
            {["title", "type", "truth", "lifecycle", "freshness", "domain", "entity"].map((k) => (
              <th key={k} className="whitespace-nowrap px-2 py-1 font-medium text-neutral-700">
                {k}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((e) => {
            const cited = citedSet.has(e.entity_id);
            return (
              <tr key={e.entity_id} className={`border-t border-neutral-100 ${cited ? "bg-amber-50" : ""}`}>
                <td className="whitespace-nowrap px-2 py-1">
                  <Link href={`/entities/${e.entity_id}`} className="text-blue-700 underline">
                    {e.title}
                  </Link>
                </td>
                <td className="whitespace-nowrap px-2 py-1">{e.entity_type}</td>
                <td className="whitespace-nowrap px-2 py-1">{e.truth_mode}</td>
                <td className="whitespace-nowrap px-2 py-1">{e.lifecycle_state}</td>
                <td className="whitespace-nowrap px-2 py-1">{e.freshness_status}</td>
                <td className="whitespace-nowrap px-2 py-1 font-mono text-[10px]">{e.domain_id.slice(0, 8)}…</td>
                <td className="whitespace-nowrap px-2 py-1">
                  <TrustExplanationDrawer
                    label="Why?"
                    meta={{
                      truthMode: e.truth_mode,
                      lifecycleState: e.lifecycle_state,
                      freshnessStatus: e.freshness_status,
                      entityType: e.entity_type,
                      domainId: e.domain_id,
                    }}
                  />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
