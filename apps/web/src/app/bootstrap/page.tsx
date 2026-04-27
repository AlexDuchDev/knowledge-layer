"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiBase, apiJson, formatApiClientError } from "@/lib/api";
import type { InstanceStatus } from "@/lib/instanceStatus";

export default function BootstrapPage() {
  const [status, setStatus] = useState<InstanceStatus | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [adminEmail, setAdminEmail] = useState("");
  const [adminName, setAdminName] = useState("");
  const [adminPassword, setAdminPassword] = useState("");
  const [domainName, setDomainName] = useState("");

  useEffect(() => {
    (async () => {
      try {
        setStatus(await apiJson<InstanceStatus>("/instance/status"));
      } catch (e) {
        setErr(formatApiClientError(e));
      }
    })();
  }, []);

  if (err && !status) {
    return (
      <main className="mx-auto max-w-lg px-6 py-16">
        <p className="text-red-700">{err}</p>
      </main>
    );
  }

  if (!status) {
    return <main className="p-8 text-sm text-neutral-600">Loading…</main>;
  }

  if (!status.needs_bootstrap) {
    return (
      <main className="mx-auto max-w-lg px-6 py-16">
        <h1 className="text-xl font-semibold">Already initialized</h1>
        <p className="mt-2 text-sm text-neutral-600">This instance already has {status.domain_count} domain(s).</p>
        <div className="mt-6 space-y-2 text-sm">
          <Link href="/" className="block text-blue-700 underline">
            Home
          </Link>
          <Link href="/source-feeds?from=cp" className="block text-blue-700 underline">
            Connect a source (wizard)
          </Link>
          <Link href="/help/getting-started" className="block text-blue-700 underline">
            Getting started guide
          </Link>
        </div>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-lg px-6 py-16">
      <h1 className="text-xl font-semibold">Create first workspace</h1>
      <p className="mt-2 text-sm text-neutral-600">
        One-time setup when the database has no domains. Creates an admin user, a domain, grants, and global admin role. Auth mode:{" "}
        <code className="rounded bg-neutral-100 px-1">{status.auth_mode}</code>
      </p>
      <p className="mt-2 text-xs text-neutral-500">
        After bootstrap, use the{" "}
        <Link href="/control-plane/setup" className="text-blue-700 underline">
          control plane setup hub
        </Link>{" "}
        for onboarding templates and launch preview (scaffold).
      </p>
      {err ? <p className="mt-4 text-sm text-red-700">{err}</p> : null}
      <form
        className="mt-6 space-y-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setErr(null);
          setBusy(true);
          try {
            await apiJson("/instance/bootstrap", {
              method: "POST",
              body: JSON.stringify({
                admin_email: adminEmail.trim(),
                admin_name: adminName.trim(),
                admin_password: adminPassword,
                domain_name: domainName.trim(),
              }),
            });
            window.location.href = "/?welcome=1";
          } catch (x) {
            setErr(x instanceof Error ? x.message : String(x));
          } finally {
            setBusy(false);
          }
        }}
      >
        <input
          required
          type="email"
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Admin email"
          value={adminEmail}
          onChange={(e) => setAdminEmail(e.target.value)}
        />
        <input
          required
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Admin display name"
          value={adminName}
          onChange={(e) => setAdminName(e.target.value)}
        />
        <input
          required
          type="password"
          minLength={8}
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Admin password (min 8)"
          value={adminPassword}
          onChange={(e) => setAdminPassword(e.target.value)}
        />
        <input
          required
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Workspace / domain name"
          value={domainName}
          onChange={(e) => setDomainName(e.target.value)}
        />
        <button
          type="submit"
          disabled={busy}
          className="w-full rounded-md bg-neutral-900 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {busy ? "Working…" : "Bootstrap instance"}
        </button>
      </form>
      <p className="mt-6 text-xs text-neutral-500">
        After bootstrap, use <Link href="/login" className="text-blue-700 underline">Sign in</Link> if{" "}
        <code className="rounded bg-neutral-100 px-1">AUTH_MODE=session</code>, or dev header for local.
      </p>
      <p className="mt-2 text-xs text-neutral-500">
        API {apiBase()}
      </p>
    </main>
  );
}
