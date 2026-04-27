"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { apiBase, apiJson, isDevPrincipalHeader } from "@/lib/api";

function LoginForm() {
  const searchParams = useSearchParams();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (isDevPrincipalHeader()) {
    return (
      <main className="mx-auto max-w-md px-6 py-16">
        <h1 className="text-xl font-semibold">Login</h1>
        <p className="mt-2 text-sm text-neutral-600">
          Session login is disabled while <code className="rounded bg-neutral-100 px-1">NEXT_PUBLIC_USE_DEV_HEADER=true</code>. The app uses the seed
          principal header instead. Set <code className="rounded bg-neutral-100 px-1">NEXT_PUBLIC_USE_DEV_HEADER=false</code> and configure{" "}
          <code className="rounded bg-neutral-100 px-1">AUTH_MODE=session</code> on the API to use email/password.
        </p>
        <Link href="/" className="mt-6 inline-block text-sm text-blue-700 underline">
          Home
        </Link>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-md px-6 py-16">
      <h1 className="text-xl font-semibold">Sign in</h1>
      <p className="mt-1 text-sm text-neutral-600">API {apiBase()}</p>
      {err ? <p className="mt-4 text-sm text-red-700">{err}</p> : null}
      <form
        className="mt-6 space-y-3"
        onSubmit={async (e) => {
          e.preventDefault();
          setErr(null);
          setBusy(true);
          try {
            await apiJson("/auth/login", {
              method: "POST",
              body: JSON.stringify({ email: email.trim(), password }),
            });
            const next = searchParams.get("next");
            window.location.href = next && next.startsWith("/") ? next : "/";
          } catch (x) {
            setErr(x instanceof Error ? x.message : String(x));
          } finally {
            setBusy(false);
          }
        }}
      >
        <input
          type="email"
          required
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <input
          type="password"
          required
          className="w-full rounded border px-3 py-2 text-sm"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <button
          type="submit"
          disabled={busy}
          className="w-full rounded-md bg-neutral-900 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {busy ? "Signing in…" : "Sign in"}
        </button>
      </form>
      <p className="mt-6 text-sm text-neutral-600">
        No account? Access is by invitation only — check your email for a link or ask an administrator.
      </p>
      <Link href="/" className="mt-4 inline-block text-sm text-blue-700 underline">
        Home
      </Link>
    </main>
  );
}

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <main className="mx-auto max-w-md px-6 py-16">
          <p className="text-sm text-neutral-600">Loading…</p>
        </main>
      }
    >
      <LoginForm />
    </Suspense>
  );
}
