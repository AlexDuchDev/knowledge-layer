"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiBase, apiJson, isDevPrincipalHeader } from "@/lib/api";

type Instance = {
  auth_mode: string;
  build_version: string;
  app_public_url: string;
  domain_count: number;
  smtp_configured: boolean;
  allow_self_registration: boolean;
  config_reference: string;
};

export default function SettingsPage() {
  const [data, setData] = useState<Instance | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [mailTo, setMailTo] = useState("");
  const [mailMsg, setMailMsg] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        setData(await apiJson<Instance>("/settings/instance"));
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
      }
    })();
  }, []);

  return (
    <main className="mx-auto max-w-2xl px-6 py-10">
      <div className="mb-8 flex items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Instance settings</h1>
          <p className="mt-1 text-sm text-neutral-600">
            Read-only deployment info and mail test. Sensitive values stay in environment variables.
          </p>
        </div>
        <Link href="/" className="text-sm text-blue-700 underline">
          Home
        </Link>
      </div>
      {err ? <div className="mb-4 rounded border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}
      {data ? (
        <dl className="space-y-3 text-sm">
          <div>
            <dt className="font-medium text-neutral-700">Auth mode</dt>
            <dd className="text-neutral-600">{data.auth_mode}</dd>
          </div>
          <div>
            <dt className="font-medium text-neutral-700">Build version</dt>
            <dd className="text-neutral-600">{data.build_version}</dd>
          </div>
          <div>
            <dt className="font-medium text-neutral-700">App public URL (invitation links)</dt>
            <dd className="text-neutral-600">{data.app_public_url}</dd>
          </div>
          <div>
            <dt className="font-medium text-neutral-700">Domains</dt>
            <dd className="text-neutral-600">{data.domain_count}</dd>
          </div>
          <div>
            <dt className="font-medium text-neutral-700">SMTP configured</dt>
            <dd className="text-neutral-600">{data.smtp_configured ? "yes" : "no (emails logged to API console)"}</dd>
          </div>
          <div>
            <dt className="font-medium text-neutral-700">Self registration</dt>
            <dd className="text-neutral-600">{data.allow_self_registration ? "enabled" : "disabled (invite only)"}</dd>
          </div>
          <div>
            <dt className="font-medium text-neutral-700">Documentation</dt>
            <dd className="text-neutral-600">{data.config_reference}</dd>
          </div>
        </dl>
      ) : !err ? (
        <p className="text-sm text-neutral-600">Loading…</p>
      ) : null}

      <section className="mt-10 border-t border-neutral-200 pt-8">
        <h2 className="text-sm font-medium">Test email</h2>
        <p className="mt-1 text-xs text-neutral-600">Sends a plain-text message via configured SMTP (or logs it).</p>
        <div className="mt-3 flex flex-wrap gap-2">
          <input
            className="min-w-[200px] flex-1 rounded border px-2 py-1 text-sm"
            placeholder="recipient@example.com"
            value={mailTo}
            onChange={(e) => setMailTo(e.target.value)}
          />
          <button
            type="button"
            className="rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white"
            onClick={async () => {
              setMailMsg(null);
              try {
                await apiJson("/settings/test-mail", { method: "POST", body: JSON.stringify({ to: mailTo.trim() }) });
                setMailMsg("Sent (or logged).");
              } catch (e) {
                setMailMsg(e instanceof Error ? e.message : String(e));
              }
            }}
          >
            Send test
          </button>
        </div>
        {mailMsg ? <p className="mt-2 text-xs text-neutral-700">{mailMsg}</p> : null}
      </section>

      <section className="mt-10 border-t border-neutral-200 pt-8">
        <h2 className="text-sm font-medium">Domain setup kits</h2>
        <p className="mt-1 text-xs text-neutral-600">
          Guided defaults for new domains (audit event only; apply roles and feeds manually). Requires publish on the target domain.
        </p>
        <DomainKitsControls />
      </section>

      <section className="mt-8 rounded-lg border border-amber-100 bg-amber-50 p-4 text-xs text-amber-950">
        <strong>Env vs UI:</strong> Database URL, session secret, TLS termination, and OIDC (future) are set at deploy time. Users, domains,
        connectors, and invitations are managed in the app. Dev header:{" "}
        <code className="rounded bg-white/80 px-1">{isDevPrincipalHeader() ? "on" : "off"}</code> · API {apiBase()}
      </section>
    </main>
  );
}

type Kit = {
  id: string;
  title: string;
  description: string;
  recommended_roles: string[];
  source_feed_hints: string[];
  job_template_ids: string[];
  governance_notes: string;
};

function DomainKitsControls() {
  const [kits, setKits] = useState<Kit[] | null>(null);
  const [domains, setDomains] = useState<{ id: string; name: string }[] | null>(null);
  const [domainId, setDomainId] = useState("");
  const [kitId, setKitId] = useState("");
  const [msg, setMsg] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const [k, d] = await Promise.all([
          apiJson<Kit[]>("/onboarding/domain-kits"),
          apiJson<{ id: string; name: string }[]>("/domains"),
        ]);
        setKits(k);
        setDomains(d);
      } catch {
        setKits([]);
        setDomains([]);
      }
    })();
  }, []);

  return (
    <div className="mt-3 space-y-3 text-sm">
      <button
        type="button"
        className="text-xs text-blue-800 underline"
        onClick={() =>
          void (async () => {
            try {
              setKits(await apiJson<Kit[]>("/onboarding/domain-kits"));
              setDomains(await apiJson<{ id: string; name: string }[]>("/domains"));
            } catch {
              /* ignore */
            }
          })()
        }
      >
        Refresh kits / domains
      </button>
      {kits && kits.length > 0 ? (
        <ul className="space-y-2 text-xs text-neutral-700">
          {kits.map((k) => (
            <li key={k.id} className="rounded border border-neutral-100 bg-neutral-50 p-2">
              <strong className="text-neutral-900">{k.title}</strong> — {k.description}
              <div className="mt-1">Roles: {k.recommended_roles.join(", ")}</div>
              <div>Feeds: {k.source_feed_hints.join("; ")}</div>
              <div>Jobs: {k.job_template_ids.join(", ")}</div>
              <div className="mt-1 italic">{k.governance_notes}</div>
            </li>
          ))}
        </ul>
      ) : null}
      <div className="flex flex-wrap items-end gap-2">
        <div>
          <div className="text-xs font-medium text-neutral-700">Domain</div>
          <select className="rounded border px-2 py-1 text-sm" value={domainId} onChange={(e) => setDomainId(e.target.value)}>
            <option value="">Select…</option>
            {(domains ?? []).map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </div>
        <div>
          <div className="text-xs font-medium text-neutral-700">Kit</div>
          <select className="rounded border px-2 py-1 text-sm" value={kitId} onChange={(e) => setKitId(e.target.value)}>
            <option value="">Select…</option>
            {(kits ?? []).map((k) => (
              <option key={k.id} value={k.id}>
                {k.title}
              </option>
            ))}
          </select>
        </div>
        <button
          type="button"
          className="rounded-md bg-neutral-900 px-3 py-1.5 text-xs text-white disabled:opacity-50"
          disabled={!domainId || !kitId}
          onClick={() =>
            void (async () => {
              setMsg(null);
              try {
                await apiJson(`/domains/${domainId}/apply-setup-kit`, {
                  method: "POST",
                  body: JSON.stringify({ kit_id: kitId }),
                });
                setMsg("Applied (logged to audit). Configure roles and feeds in Access / Source feeds.");
              } catch (e) {
                setMsg(e instanceof Error ? e.message : String(e));
              }
            })()
          }
        >
          Apply kit (audit)
        </button>
      </div>
      {msg ? <p className="text-xs text-neutral-700">{msg}</p> : null}
    </div>
  );
}
