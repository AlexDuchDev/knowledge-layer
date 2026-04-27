"use client";

import { useCallback, useMemo, useState } from "react";
import Link from "next/link";
import { apiBase, apiJson, isDevPrincipalHeader, principalUserId } from "@/lib/api";

type Json = Record<string, unknown>;

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

export default function AccessPage() {
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const run = useCallback(async (key: string, fn: () => Promise<void>) => {
    setErr(null);
    setBusy(key);
    try {
      await fn();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }, []);

  return (
    <main className="mx-auto max-w-5xl px-6 py-10">
      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Users and Access</h1>
          <p className="mt-1 text-sm text-neutral-600">
            Identity admin: users, domain grants, role bindings. API{" "}
            <code className="rounded bg-neutral-100 px-1">{apiBase()}</code> · principal{" "}
            <code className="rounded bg-neutral-100 px-1">{principalUserId()}</code>
          </p>
        </div>
        <Link href="/" className="text-sm text-blue-700 underline">
          Home
        </Link>
      </div>

      <nav className="mb-8 flex flex-wrap gap-2 border-b border-neutral-200 pb-3 text-sm" aria-label="Access sections">
        <a href="#users" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900 hover:bg-neutral-50">
          Users
        </a>
        <a href="#grants" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900 hover:bg-neutral-50">
          Domain grants
        </a>
        <a href="#roles" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900 hover:bg-neutral-50">
          Role bindings
        </a>
        <a href="#teams" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900 hover:bg-neutral-50">
          Teams
        </a>
        <a href="#import" className="rounded border border-neutral-200 bg-white px-3 py-1.5 text-neutral-900 hover:bg-neutral-50">
          Bulk import
        </a>
        <a href="#api-keys" className="rounded border border-dashed border-neutral-300 bg-neutral-50 px-3 py-1.5 text-neutral-600">
          API keys (planned)
        </a>
      </nav>

      {err ? (
        <div className="mb-6 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div>
      ) : null}

      <div className="flex flex-col gap-12">
        <UsersSection run={run} busy={busy} />
        <DomainGrantsSection run={run} busy={busy} />
        <RoleBindingsSection run={run} busy={busy} />
        <TeamsSection run={run} busy={busy} />
        <BulkImportSection />
        <section id="api-keys" className="border-t border-neutral-200 pt-10">
          <h2 className="text-lg font-medium">API keys</h2>
          <p className="mt-1 text-sm text-neutral-600">
            Automation principals and scoped API keys are not exposed in this pilot UI yet. When enabled, they will appear here per{" "}
            <span className="font-medium">Access &amp; trust</span> IA.
          </p>
        </section>
      </div>
    </main>
  );
}

function BulkImportSection() {
  const [msg, setMsg] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  return (
    <section id="import" className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Bulk user import</h2>
      <p className="mt-1 text-sm text-neutral-600" title="CSV columns: email, domain_id required; optional name, access_level, sensitivity_cap, role_id">
        Upload CSV (<code className="rounded bg-neutral-100 px-1">email</code>, <code className="rounded bg-neutral-100 px-1">domain_id</code> required).
        Mode <strong>invite</strong> creates invitations; <strong>active</strong> creates users immediately.
      </p>
      <form
        className="mt-4 flex max-w-xl flex-col gap-2"
        onSubmit={async (e) => {
          e.preventDefault();
          setMsg(null);
          const fd = new FormData(e.currentTarget);
          setBusy(true);
          try {
            const headers: Record<string, string> = {};
            if (isDevPrincipalHeader()) {
              headers["X-Principal-User-ID"] = principalUserId();
            }
            const res = await fetch(`${apiBase()}/users/import`, {
              method: "POST",
              credentials: "include",
              headers,
              body: fd,
            });
            const text = await res.text();
            if (!res.ok) throw new Error(text.slice(0, 400));
            setMsg(text.slice(0, 2000));
          } catch (err) {
            setMsg(err instanceof Error ? err.message : String(err));
          } finally {
            setBusy(false);
          }
        }}
      >
        <input type="file" name="file" accept=".csv,text/csv" required className="text-sm" />
        <select name="mode" className="rounded border px-2 py-1 text-sm">
          <option value="invite">invite</option>
          <option value="active">active</option>
        </select>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" name="send_invites" value="true" />
          Send invitation emails (SMTP must be configured)
        </label>
        <button type="submit" disabled={busy} className="w-fit rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50">
          {busy ? "Uploading…" : "POST /users/import"}
        </button>
      </form>
      {msg ? (
        <pre className="mt-3 max-h-64 overflow-auto rounded bg-neutral-50 p-3 text-xs text-neutral-800">{msg}</pre>
      ) : null}
    </section>
  );
}

function UsersSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [users, setUsers] = useState<Json[] | null>(null);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [selectedId, setSelectedId] = useState("");

  const selected = useMemo(() => {
    if (!selectedId || !users) return null;
    return users.find((u) => asStr(u.id) === selectedId) ?? null;
  }, [selectedId, users]);

  return (
    <section id="users" className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Users</h2>
      <p className="mt-1 text-sm text-neutral-600" title="Production: prefer invitations (POST /invitations) or CSV import so users set their own passwords.">
        List users and create accounts (requires publish on a granted domain). For production, use invitations instead of shared passwords.
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("users", async () => {
              const list = await apiJson<Json[]>("/users");
              setUsers(list);
            })
          }
        >
          {busy === "users" ? "Loading…" : "Refresh users"}
        </button>
      </div>
      {users ? (
        <div className="mt-4">
          <label className="text-sm font-medium text-neutral-700">Selected user (for grants / bindings)</label>
          <select
            className="mt-1 block w-full max-w-xl rounded-md border border-neutral-300 px-3 py-2 text-sm"
            value={selectedId}
            onChange={(e) => setSelectedId(e.target.value)}
          >
            <option value="">— choose —</option>
            {users.map((u) => (
              <option key={asStr(u.id)} value={asStr(u.id)}>
                {asStr(u.email)} ({asStr(u.name)})
              </option>
            ))}
          </select>
          {selected ? (
            <div className="mt-3 space-y-2">
              <Link
                href={`/control-plane/users/${encodeURIComponent(asStr(selected.id))}`}
                className="inline-block text-sm text-blue-700 underline"
              >
                Open user detail
              </Link>
              <pre className="max-h-40 overflow-auto rounded-md bg-neutral-50 p-3 text-xs">{JSON.stringify(selected, null, 2)}</pre>
            </div>
          ) : null}
        </div>
      ) : null}

      <div className="mt-8 rounded-lg border border-neutral-200 p-4">
        <h3 className="text-sm font-medium">Create user</h3>
        <div className="mt-3 flex max-w-xl flex-col gap-2 sm:flex-row">
          <input
            className="flex-1 rounded-md border border-neutral-300 px-3 py-2 text-sm"
            placeholder="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <input
            className="flex-1 rounded-md border border-neutral-300 px-3 py-2 text-sm"
            placeholder="display name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <button
          type="button"
          className="mt-3 rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null || !email.trim() || !name.trim()}
          onClick={() =>
            run("create-user", async () => {
              await apiJson("/users", {
                method: "POST",
                body: JSON.stringify({ email: email.trim(), name: name.trim() }),
              });
              setEmail("");
              setName("");
              const list = await apiJson<Json[]>("/users");
              setUsers(list);
            })
          }
        >
          {busy === "create-user" ? "Creating…" : "POST /users"}
        </button>
      </div>
    </section>
  );
}

function DomainGrantsSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [domains, setDomains] = useState<Json[] | null>(null);
  const [grants, setGrants] = useState<Json[] | null>(null);
  const [userId, setUserId] = useState("");
  const [domainId, setDomainId] = useState("");
  const [accessLevel, setAccessLevel] = useState("read");
  const [sensitivityCap, setSensitivityCap] = useState("50");

  return (
    <section id="grants" className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Domain grants</h2>
      <p className="mt-1 text-sm text-neutral-600">
        Requires publish on the target domain. Upsert uses (user_id, domain_id) conflict.
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("domains", async () => {
              const list = await apiJson<Json[]>("/domains");
              setDomains(list);
            })
          }
        >
          {busy === "domains" ? "Loading…" : "Load domains"}
        </button>
      </div>
      <div className="mt-6 grid max-w-xl gap-3">
        <input
          className="rounded-md border border-neutral-300 px-3 py-2 text-sm"
          placeholder="user_id (UUID)"
          value={userId}
          onChange={(e) => setUserId(e.target.value)}
        />
        <select
          className="rounded-md border border-neutral-300 px-3 py-2 text-sm"
          value={domainId}
          onChange={(e) => setDomainId(e.target.value)}
        >
          <option value="">— domain —</option>
          {(domains ?? []).map((d) => (
            <option key={asStr(d.id)} value={asStr(d.id)}>
              {asStr(d.name)} ({asStr(d.id).slice(0, 8)}…)
            </option>
          ))}
        </select>
        <div className="flex gap-2">
          <select
            className="flex-1 rounded-md border border-neutral-300 px-3 py-2 text-sm"
            value={accessLevel}
            onChange={(e) => setAccessLevel(e.target.value)}
          >
            <option value="read">read</option>
            <option value="write">write</option>
            <option value="admin">admin</option>
          </select>
          <input
            className="w-28 rounded-md border border-neutral-300 px-3 py-2 text-sm"
            type="number"
            value={sensitivityCap}
            onChange={(e) => setSensitivityCap(e.target.value)}
          />
        </div>
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null || !userId.trim() || !domainId}
          onClick={() =>
            run("grant", async () => {
              await apiJson("/domain-grants", {
                method: "POST",
                body: JSON.stringify({
                  user_id: userId.trim(),
                  domain_id: domainId,
                  access_level: accessLevel,
                  sensitivity_cap: Number(sensitivityCap) || 0,
                }),
              });
              const g = await apiJson<Json[]>(`/domain-grants?user_id=${encodeURIComponent(userId.trim())}`);
              setGrants(g);
            })
          }
        >
          {busy === "grant" ? "Saving…" : "POST /domain-grants"}
        </button>
        <button
          type="button"
          className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy !== null || !userId.trim()}
          onClick={() =>
            run("list-grants", async () => {
              const g = await apiJson<Json[]>(`/domain-grants?user_id=${encodeURIComponent(userId.trim())}`);
              setGrants(g);
            })
          }
        >
          {busy === "list-grants" ? "Loading…" : "GET domain-grants for user"}
        </button>
      </div>
      {grants ? (
        <ul className="mt-4 space-y-2 text-sm">
          {grants.map((g) => (
            <li key={asStr(g.id)} className="rounded-md border border-neutral-100 bg-neutral-50 px-3 py-2 font-mono text-xs">
              domain {asStr(g.domain_id)} · {asStr(g.access_level)} · cap {asStr(g.sensitivity_cap)}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function RoleBindingsSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [roles, setRoles] = useState<Json[] | null>(null);
  const [bindings, setBindings] = useState<Json[] | null>(null);
  const [userId, setUserId] = useState("");
  const [roleId, setRoleId] = useState("");
  const [scopeType, setScopeType] = useState("global");
  const [scopeId, setScopeId] = useState("");

  return (
    <section id="roles" className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Role bindings</h2>
      <p className="mt-1 text-sm text-neutral-600">Global or domain-scoped bindings. Domain scope requires publish on that domain.</p>
      <div className="mt-4">
        <button
          type="button"
          className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("roles", async () => {
              const list = await apiJson<Json[]>("/roles");
              setRoles(list);
            })
          }
        >
          {busy === "roles" ? "Loading…" : "Load roles"}
        </button>
      </div>
      <div className="mt-6 grid max-w-xl gap-3">
        <input
          className="rounded-md border border-neutral-300 px-3 py-2 text-sm"
          placeholder="user_id (UUID)"
          value={userId}
          onChange={(e) => setUserId(e.target.value)}
        />
        <select
          className="rounded-md border border-neutral-300 px-3 py-2 text-sm"
          value={roleId}
          onChange={(e) => setRoleId(e.target.value)}
        >
          <option value="">— role —</option>
          {(roles ?? []).map((r) => (
            <option key={asStr(r.id)} value={asStr(r.id)}>
              {asStr(r.name)}
            </option>
          ))}
        </select>
        <select
          className="rounded-md border border-neutral-300 px-3 py-2 text-sm"
          value={scopeType}
          onChange={(e) => setScopeType(e.target.value)}
        >
          <option value="global">global</option>
          <option value="domain">domain</option>
        </select>
        {scopeType === "domain" ? (
          <input
            className="rounded-md border border-neutral-300 px-3 py-2 text-sm"
            placeholder="scope_id (domain UUID)"
            value={scopeId}
            onChange={(e) => setScopeId(e.target.value)}
          />
        ) : null}
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null || !userId.trim() || !roleId || (scopeType === "domain" && !scopeId.trim())}
          onClick={() =>
            run("bind", async () => {
              const body: Json = {
                user_id: userId.trim(),
                role_id: roleId,
                scope_type: scopeType,
              };
              if (scopeType === "domain") body.scope_id = scopeId.trim();
              await apiJson("/user-role-bindings", { method: "POST", body: JSON.stringify(body) });
              const b = await apiJson<Json[]>(`/user-role-bindings?user_id=${encodeURIComponent(userId.trim())}`);
              setBindings(b);
            })
          }
        >
          {busy === "bind" ? "Saving…" : "POST /user-role-bindings"}
        </button>
        <button
          type="button"
          className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy !== null || !userId.trim()}
          onClick={() =>
            run("list-bind", async () => {
              const b = await apiJson<Json[]>(`/user-role-bindings?user_id=${encodeURIComponent(userId.trim())}`);
              setBindings(b);
            })
          }
        >
          {busy === "list-bind" ? "Loading…" : "GET bindings for user"}
        </button>
      </div>
      {bindings ? (
        <ul className="mt-4 space-y-2 text-sm">
          {bindings.map((b) => (
            <li key={asStr(b.id)} className="rounded-md border border-neutral-100 bg-neutral-50 px-3 py-2 font-mono text-xs">
              role {asStr(b.role_id)} · {asStr(b.scope_type)}
              {b.scope_id ? ` · ${asStr(b.scope_id)}` : ""}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function TeamsSection({
  run,
  busy,
}: {
  run: (k: string, fn: () => Promise<void>) => Promise<void>;
  busy: string | null;
}) {
  const [teams, setTeams] = useState<Json[] | null>(null);
  const [tname, setTname] = useState("");
  const [userId, setUserId] = useState("");
  const [teamId, setTeamId] = useState("");
  const [mtype, setMtype] = useState("member");

  return (
    <section id="teams" className="border-t border-neutral-200 pt-10">
      <h2 className="text-lg font-medium">Teams & memberships</h2>
      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-md border border-neutral-300 px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy !== null}
          onClick={() =>
            run("teams", async () => {
              const list = await apiJson<Json[]>("/teams?limit=100");
              setTeams(list);
            })
          }
        >
          {busy === "teams" ? "Loading…" : "List teams"}
        </button>
      </div>
      {teams ? (
        <ul className="mt-3 space-y-1 text-sm text-neutral-700">
          {teams.map((t) => (
            <li key={asStr(t.id)}>
              {asStr(t.name)} <span className="font-mono text-xs text-neutral-500">{asStr(t.id)}</span>
            </li>
          ))}
        </ul>
      ) : null}
      <div className="mt-6 rounded-lg border border-neutral-200 p-4">
        <h3 className="text-sm font-medium">Create team</h3>
        <input
          className="mt-2 block max-w-md rounded-md border border-neutral-300 px-3 py-2 text-sm"
          placeholder="Team name"
          value={tname}
          onChange={(e) => setTname(e.target.value)}
        />
        <button
          type="button"
          className="mt-2 rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null || !tname.trim()}
          onClick={() =>
            run("mkteam", async () => {
              await apiJson("/teams", { method: "POST", body: JSON.stringify({ name: tname.trim() }) });
              setTname("");
              setTeams(await apiJson<Json[]>("/teams?limit=100"));
            })
          }
        >
          {busy === "mkteam" ? "Creating…" : "POST /teams"}
        </button>
      </div>
      <div className="mt-6 rounded-lg border border-neutral-200 p-4">
        <h3 className="text-sm font-medium">Add team membership</h3>
        <div className="mt-2 flex max-w-xl flex-col gap-2 sm:flex-row">
          <input
            className="flex-1 rounded-md border border-neutral-300 px-3 py-2 text-sm"
            placeholder="user_id"
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
          />
          <input
            className="flex-1 rounded-md border border-neutral-300 px-3 py-2 text-sm"
            placeholder="team_id"
            value={teamId}
            onChange={(e) => setTeamId(e.target.value)}
          />
        </div>
        <select
          className="mt-2 rounded-md border border-neutral-300 px-3 py-2 text-sm"
          value={mtype}
          onChange={(e) => setMtype(e.target.value)}
        >
          <option value="member">member</option>
          <option value="admin">admin</option>
        </select>
        <button
          type="button"
          className="mt-2 rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50"
          disabled={busy !== null || !userId.trim() || !teamId.trim()}
          onClick={() =>
            run("member", async () => {
              await apiJson("/user-team-memberships", {
                method: "POST",
                body: JSON.stringify({
                  user_id: userId.trim(),
                  team_id: teamId.trim(),
                  membership_type: mtype,
                }),
              });
            })
          }
        >
          {busy === "member" ? "Saving…" : "POST /user-team-memberships"}
        </button>
      </div>
    </section>
  );
}
