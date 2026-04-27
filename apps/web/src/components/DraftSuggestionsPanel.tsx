"use client";

import { useState } from "react";
import { apiJson } from "@/lib/api";

export function DraftSuggestionsPanel({ entityId }: { entityId: string }) {
  const [text, setText] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  return (
    <section className="mt-6 rounded-lg border border-amber-200 bg-amber-50/40 p-4">
      <h2 className="text-sm font-semibold text-neutral-900">AI draft suggestions</h2>
      <p className="mt-1 text-xs text-neutral-700">
        Suggestions only — you copy or edit manually. Does not write to the server or change lifecycle. Draft entities only.
      </p>
      <button
        type="button"
        disabled={busy}
        className="mt-3 rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-xs font-medium text-neutral-900 disabled:opacity-50"
        onClick={() => {
          void (async () => {
            setBusy(true);
            setErr(null);
            try {
              const out = await apiJson<{ suggestions_markdown: string; disclaimer?: string }>("/ai/draft-suggestions", {
                method: "POST",
                body: JSON.stringify({ entity_id: entityId }),
              });
              setText(out.suggestions_markdown);
            } catch (e) {
              setErr(e instanceof Error ? e.message : String(e));
            } finally {
              setBusy(false);
            }
          })();
        }}
      >
        {busy ? "Requesting…" : "Get suggestions"}
      </button>
      {err ? <p className="mt-2 text-xs text-red-800">{err}</p> : null}
      {text ? (
        <div className="mt-3 rounded border border-neutral-200 bg-white p-3">
          <pre className="max-h-[360px] overflow-auto whitespace-pre-wrap text-xs text-neutral-800">{text}</pre>
          <button
            type="button"
            className="mt-2 text-xs text-blue-800 underline"
            onClick={() => void navigator.clipboard?.writeText(text)}
          >
            Copy suggestions
          </button>
        </div>
      ) : null}
    </section>
  );
}
