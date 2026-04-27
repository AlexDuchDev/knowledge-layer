"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { AppBreadcrumb } from "@/components/AppBreadcrumb";
import { apiJson } from "@/lib/api";

type User = {
  id: string;
  email: string;
  name: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export default function AdminUserDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const [u, setU] = useState<User | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    void (async () => {
      try {
        setU(await apiJson<User>(`/users/${encodeURIComponent(id)}`));
        setErr(null);
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
        setU(null);
      }
    })();
  }, [id]);

  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <AppBreadcrumb
        items={[
          { label: "Home", href: "/" },
          { label: "Users", href: "/control-plane/users" },
          { label: u?.name || u?.email || "User" },
        ]}
      />
      <h1 className="text-2xl font-semibold tracking-tight">User</h1>
      {err ? <div className="my-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      {u ? (
        <dl className="mt-6 space-y-2 text-sm">
          <div>
            <dt className="text-neutral-500">Email</dt>
            <dd className="font-medium">{u.email}</dd>
          </div>
          <div>
            <dt className="text-neutral-500">Name</dt>
            <dd>{u.name}</dd>
          </div>
          <div>
            <dt className="text-neutral-500">Status</dt>
            <dd>{u.status}</dd>
          </div>
          <div>
            <dt className="text-neutral-500">Id</dt>
            <dd className="font-mono text-xs">{u.id}</dd>
          </div>
        </dl>
      ) : null}
      <p className="mt-8">
        <Link href="/control-plane/users" className="text-sm text-blue-700 underline">
          Back to access & trust
        </Link>
      </p>
    </main>
  );
}
