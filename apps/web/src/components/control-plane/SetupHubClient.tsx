"use client";

/**
 * Native CP setup hub (Phase 2.1.5 — replaces /admin/setup with CP-native).
 *
 * Shows seeded onboarding templates and the principal's recent setup sessions.
 * "New session" creates a session via POST /api/onboarding/sessions and routes
 * the user into the wizard at /control-plane/setup/session/[id].
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { apiJson, formatApiClientError } from "@/lib/api";

type Template = {
  id: string;
  code: string;
  title: string;
  description: string;
};

type SessionSummary = {
  id: string;
  status: string;
  template_code?: string | null;
  updated_at: string;
};

export function SetupHubClient() {
  const router = useRouter();
  const [err, setErr] = useState<string | null>(null);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setBusy(true);
    setErr(null);
    try {
      const [t, s] = await Promise.all([
        apiJson<Template[]>("/api/onboarding/templates"),
        apiJson<SessionSummary[]>("/api/onboarding/sessions?limit=50"),
      ]);
      setTemplates(t);
      setSessions(s);
    } catch (e) {
      setErr(formatApiClientError(e));
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function createSession() {
    setBusy(true);
    setErr(null);
    try {
      const sess = await apiJson<{ id: string }>("/api/onboarding/sessions", { method: "POST" });
      router.push(`/control-plane/setup/session/${encodeURIComponent(sess.id)}`);
    } catch (e) {
      setErr(formatApiClientError(e));
      setBusy(false);
    }
  }

  return (
    <div className="mt-6 space-y-8">
      {err ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{err}</div>
      ) : null}

      <div className="flex flex-wrap gap-3">
        <button onClick={createSession} disabled={busy} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-400">
          {busy ? "Working…" : "New setup session"}
        </button>
        <button onClick={load} disabled={busy} className="rounded-md border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50">
          Refresh
        </button>
      </div>

      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h2 className="text-lg font-medium text-gray-900">Templates</h2>
        <p className="mt-1 text-xs text-gray-500">
          Seeded onboarding modes. Pick one inside an open session to load preset bundles + sensible defaults.
        </p>
        <ul className="mt-3 space-y-2 text-sm">
          {templates.length === 0 ? (
            <li className="text-gray-500">{busy ? "Loading…" : "No templates registered."}</li>
          ) : (
            templates.map((tpl) => (
              <li key={tpl.id} className="border-b border-gray-100 pb-2 last:border-0">
                <span className="font-mono text-xs text-gray-600">{tpl.code}</span> — <strong className="text-gray-900">{tpl.title}</strong>
                <p className="text-xs text-gray-500">{tpl.description}</p>
              </li>
            ))
          )}
        </ul>
      </section>

      <section className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <h2 className="text-lg font-medium text-gray-900">Your sessions</h2>
        <table className="mt-3 w-full text-left text-sm">
          <thead className="border-b border-gray-200 text-xs uppercase text-gray-500">
            <tr>
              <th className="py-2">Status</th>
              <th className="py-2">Template</th>
              <th className="py-2">Updated</th>
              <th className="py-2" />
            </tr>
          </thead>
          <tbody>
            {sessions.length === 0 ? (
              <tr>
                <td colSpan={4} className="py-4 text-center text-gray-500">No sessions yet — start one above.</td>
              </tr>
            ) : (
              sessions.map((s) => (
                <tr key={s.id} className="border-b border-gray-100">
                  <td className="py-2">
                    <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-800">{s.status}</span>
                  </td>
                  <td className="py-2 text-xs">{s.template_code ?? "—"}</td>
                  <td className="py-2 text-xs text-gray-600">{new Date(s.updated_at).toLocaleString()}</td>
                  <td className="py-2 text-right">
                    <Link href={`/control-plane/setup/session/${encodeURIComponent(s.id)}`} className="text-blue-700 underline hover:text-blue-900">
                      Open
                    </Link>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </section>
    </div>
  );
}
