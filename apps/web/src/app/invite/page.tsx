"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import Link from "next/link";
import { apiBase, apiJson } from "@/lib/api";

function InviteForm() {
  const sp = useSearchParams();
  const token = sp.get("token") ?? "";
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState<{ email: string } | null>(null);

  return (
    <main className="mx-auto max-w-md px-6 py-16">
      <h1 className="text-xl font-semibold">Accept invitation</h1>
      <p className="mt-1 text-sm text-neutral-600">API {apiBase()}</p>
      {!token ? (
        <p className="mt-4 text-sm text-red-700">Missing token in URL. Open the link from your invitation email.</p>
      ) : (
        <>
          <button
            type="button"
            className="mt-4 rounded border border-neutral-300 px-3 py-1.5 text-xs"
            onClick={async () => {
              setErr(null);
              try {
                const p = await apiJson<{ email: string }>(`/invitations/preview?token=${encodeURIComponent(token)}`);
                setPreview(p);
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              }
            }}
          >
            Verify token
          </button>
          {preview ? <p className="mt-2 text-sm">Email: {preview.email}</p> : null}
          {err ? <p className="mt-2 text-sm text-red-700">{err}</p> : null}
          <form
            className="mt-6 space-y-3"
            onSubmit={async (e) => {
              e.preventDefault();
              setErr(null);
              setBusy(true);
              try {
                await apiJson("/invitations/accept", {
                  method: "POST",
                  body: JSON.stringify({ token, name: name.trim(), password }),
                });
                window.location.href = "/";
              } catch (x) {
                setErr(x instanceof Error ? x.message : String(x));
              } finally {
                setBusy(false);
              }
            }}
          >
            <input
              required
              className="w-full rounded border px-3 py-2 text-sm"
              placeholder="Your name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <input
              type="password"
              required
              minLength={8}
              className="w-full rounded border px-3 py-2 text-sm"
              placeholder="Password (min 8 characters)"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <button
              type="submit"
              disabled={busy}
              className="w-full rounded-md bg-neutral-900 py-2 text-sm font-medium text-white disabled:opacity-50"
            >
              {busy ? "Creating account…" : "Activate account"}
            </button>
          </form>
        </>
      )}
      <Link href="/" className="mt-8 inline-block text-sm text-blue-700 underline">
        Home
      </Link>
    </main>
  );
}

export default function InvitePage() {
  return (
    <Suspense fallback={<p className="p-8 text-sm text-neutral-600">Loading…</p>}>
      <InviteForm />
    </Suspense>
  );
}
