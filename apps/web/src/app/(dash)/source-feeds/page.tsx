"use client";

import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { DocHelpCallout } from "@/components/guidance/DocHelpCallout";
import { FromCpBanner } from "@/components/shell/FromCpBanner";
import { apiBase, apiJson, principalUserId } from "@/lib/api";

type Json = Record<string, unknown>;

function asStr(v: unknown): string {
  return typeof v === "string" ? v : v == null ? "" : String(v);
}

function mapSourceFeedError(raw: string): string {
  const t = raw.toLowerCase();
  if (t.includes("access denied") || t.includes("403")) return "You do not have permission to manage source feeds in this domain or at this sensitivity.";
  if (t.includes("owner required")) return "Assign an owner before activation.";
  if (t.includes("allowed_job_types")) return "Select at least one allowed job type.";
  if (t.includes("ingestion_mode")) return "Choose an ingestion mode.";
  if (t.includes("invalid json")) return "Invalid request payload.";
  return raw;
}

type ConnectorKind =
  | "telegram"
  | "google_drive"
  | "jira"
  | "trello"
  | "asana"
  | "linear"
  | "slack"
  | "mattermost"
  | "notion"
  | "confluence"
  | "google_calendar"
  | "zendesk"
  | "hubspot"
  | "intercom"
  | "fireflies"
  | "oauth_blob"
  | "raw";

function mapConnectorKind(connectorType: string): ConnectorKind {
  const typ = connectorType.toLowerCase();
  if (typ.includes("jira")) return "jira";
  if (typ.includes("trello")) return "trello";
  if (typ.includes("asana")) return "asana";
  if (typ.includes("linear")) return "linear";
  if (typ.includes("slack")) return "slack";
  if (typ.includes("mattermost")) return "mattermost";
  if (typ.includes("notion")) return "notion";
  if (typ.includes("confluence")) return "confluence";
  if (typ.includes("google_calendar") || (typ.includes("calendar") && typ.includes("google"))) return "google_calendar";
  if (typ.includes("drive") || typ.includes("google_drive")) return "google_drive";
  if (typ.includes("telegram")) return "telegram";
  if (typ.includes("zendesk")) return "zendesk";
  if (typ.includes("hubspot")) return "hubspot";
  if (typ.includes("intercom")) return "intercom";
  if (typ.includes("fireflies")) return "fireflies";
  if (typ.includes("gmail") || typ.includes("microsoft") || typ.includes("outlook")) return "oauth_blob";
  if (typ.includes("google")) return "google_drive";
  return "raw";
}

function domainPayload(selDomain: string): Record<string, string> {
  const o: Record<string, string> = {};
  if (selDomain.trim()) o.domain_id = selDomain.trim();
  return o;
}

const STEPS = [
  "Choose connector",
  "Authenticate / config",
  "Map source",
  "Governance",
  "Ingestion behavior",
  "Review",
  "Create & preview",
  "Activate",
] as const;

const SENSITIVITY_LEVELS: { value: string; label: string; hint: string }[] = [
  { value: "0", label: "Public internal", hint: "Widest internal visibility appropriate for non-sensitive operational content." },
  { value: "1", label: "Team restricted", hint: "Typical default for team channels and working material." },
  { value: "2", label: "Domain restricted / leadership", hint: "Stricter than team-wide; align with domain policy." },
  { value: "3", label: "Strictly confidential", hint: "Highest sensitivity in this pilot scale; confirm with governance." },
];

const JOB_OPTIONS = [{ id: "weekly_digest", label: "Weekly digest" }];

const INGESTION_MODES: { value: string; label: string; hint: string }[] = [
  { value: "ingestion_only", label: "Raw capture / ingestion only", hint: "Conservative: store normalized evidence; API value `ingestion_only`." },
  { value: "governed_processing", label: "Governed processing", hint: "Allow indexing and structured downstream use (stored as `governed_processing` if supported)." },
  { value: "governed_processing_with_jobs", label: "Governed processing with jobs", hint: "Allows approved knowledge jobs on this feed when the platform enables it." },
];

type UserRow = { id: string; name: string; email: string };

function SourceFeedsPageInner() {
  const sp = useSearchParams();
  const [step, setStep] = useState(0);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const [feeds, setFeeds] = useState<Json[] | null>(null);
  const [domains, setDomains] = useState<{ id: string; name: string }[] | null>(null);
  const [connectors, setConnectors] = useState<Json[] | null>(null);
  const [users, setUsers] = useState<UserRow[] | null>(null);

  const [selConnector, setSelConnector] = useState("");
  const [selDomain, setSelDomain] = useState("");
  const [ownerId, setOwnerId] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [sensitivity, setSensitivity] = useState("1");
  const [ingestionMode, setIngestionMode] = useState("ingestion_only");
  const [jobs, setJobs] = useState<Record<string, boolean>>({ weekly_digest: true });

  const [connectorKind, setConnectorKind] = useState<ConnectorKind>("raw");
  const [botToken, setBotToken] = useState("");
  const [folderId, setFolderId] = useState("");
  const [saJson, setSaJson] = useState("{}");
  const [driveFolders, setDriveFolders] = useState<{ id: string; name: string }[] | null>(null);
  const [jiraSite, setJiraSite] = useState("");
  const [jiraEmail, setJiraEmail] = useState("");
  const [jiraToken, setJiraToken] = useState("");
  const [jiraProjectKey, setJiraProjectKey] = useState("");
  const [jiraProjects, setJiraProjects] = useState<{ id: string; key: string; name: string }[] | null>(null);
  const [trelloKey, setTrelloKey] = useState("");
  const [trelloToken, setTrelloToken] = useState("");
  const [trelloBoardId, setTrelloBoardId] = useState("");
  const [trelloBoards, setTrelloBoards] = useState<{ id: string; name: string }[] | null>(null);
  const [asanaPAT, setAsanaPAT] = useState("");
  const [asanaProjectGid, setAsanaProjectGid] = useState("");
  const [asanaProjects, setAsanaProjects] = useState<{ gid: string; name: string; workspace_name?: string }[] | null>(null);
  const [linearKey, setLinearKey] = useState("");
  const [linearTeamId, setLinearTeamId] = useState("");
  const [linearTeams, setLinearTeams] = useState<{ id: string; key: string; name: string }[] | null>(null);
  const [slackBotToken, setSlackBotToken] = useState("");
  const [slackChannelId, setSlackChannelId] = useState("");
  const [slackChannels, setSlackChannels] = useState<{ id: string; name: string }[] | null>(null);
  const [mmBase, setMmBase] = useState("");
  const [mmToken, setMmToken] = useState("");
  const [mmChannelId, setMmChannelId] = useState("");
  const [mmChannels, setMmChannels] = useState<{ id: string; display_name: string; team_name?: string }[] | null>(null);
  const [notionTok, setNotionTok] = useState("");
  const [notionScope, setNotionScope] = useState<"page" | "database">("page");
  const [notionPick, setNotionPick] = useState("");
  const [notionItems, setNotionItems] = useState<{ object: string; id: string; title: string }[] | null>(null);
  const [confBase, setConfBase] = useState("");
  const [confAuth, setConfAuth] = useState("");
  const [confSpaceKey, setConfSpaceKey] = useState("");
  const [confSpaces, setConfSpaces] = useState<{ key: string; name: string }[] | null>(null);
  const [calSaJson, setCalSaJson] = useState("{}");
  const [calId, setCalId] = useState("");
  const [calList, setCalList] = useState<{ id: string; summary: string }[] | null>(null);
  const [zdSub, setZdSub] = useState("");
  const [zdEmail, setZdEmail] = useState("");
  const [zdToken, setZdToken] = useState("");
  const [zdKind, setZdKind] = useState<"all" | "view">("all");
  const [zdViewId, setZdViewId] = useState("");
  const [zdViews, setZdViews] = useState<{ id: number; title: string }[] | null>(null);
  const [hsTok, setHsTok] = useState("");
  const [hsKind, setHsKind] = useState<"contacts" | "companies" | "deals">("contacts");
  const [intercomTok, setIntercomTok] = useState("");
  const [ffKey, setFfKey] = useState("");
  const [advancedJson, setAdvancedJson] = useState("{}");

  const [createdFeedId, setCreatedFeedId] = useState("");
  const [previewSummary, setPreviewSummary] = useState<string | null>(null);

  const run = useCallback(async (fn: () => Promise<void>) => {
    setErr(null);
    setBusy(true);
    try {
      await fn();
    } catch (e) {
      const raw = e instanceof Error ? e.message : String(e);
      setErr(mapSourceFeedError(raw));
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    const sid = (sp.get("oauth_sid") || "").trim();
    if (!sid) return;
    void run(async () => {
      const out = await apiJson<{ connector_config_patch: Json }>("/integrations/oauth/consume", {
        method: "POST",
        body: JSON.stringify({ oauth_sid: sid }),
      });
      setAdvancedJson((prev) => {
        let base: Json = {};
        try {
          base = JSON.parse(prev) as Json;
        } catch {
          base = {};
        }
        return JSON.stringify({ ...base, ...(out.connector_config_patch ?? {}) }, null, 2);
      });
      setStep(1);
    });
  }, [sp, run]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const [d, c] = await Promise.all([
          apiJson<{ id: string; name: string }[]>("/domains"),
          apiJson<Json[]>("/connectors"),
        ]);
        if (!cancelled) {
          setDomains(d);
          setConnectors(c);
        }
        try {
          const u = await apiJson<UserRow[]>("/users");
          if (!cancelled) setUsers(u);
        } catch {
          if (!cancelled) setUsers([]);
        }
      } catch {
        if (!cancelled) setErr("Failed to load domains or connectors.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (ownerId) return;
    const pid = principalUserId();
    if (users?.some((u) => u.id === pid)) setOwnerId(pid);
  }, [users, ownerId]);

  const selectedConnector = useMemo(() => (connectors ?? []).find((c) => asStr(c.id) === selConnector), [connectors, selConnector]);

  useEffect(() => {
    setConnectorKind(mapConnectorKind(asStr(selectedConnector?.type)));
  }, [selectedConnector]);

  const connectorDescription = useMemo(() => {
    const t = asStr(selectedConnector?.type).toLowerCase();
    if (t.includes("telegram")) return "Telegram — bot token; governed chat ingestion.";
    if (t.includes("google_drive") || (t.includes("drive") && !t.includes("calendar")))
      return "Google Drive — service account JSON; list root folders then pick folder id.";
    if (t.includes("calendar")) return "Google Calendar — service account; pick a calendar id from the list.";
    if (t.includes("jira")) return "Jira Cloud — list projects after entering site + API token.";
    if (t.includes("trello")) return "Trello — API key + token; list boards and pick one.";
    if (t.includes("asana")) return "Asana — personal access token; list projects across workspaces.";
    if (t.includes("linear")) return "Linear — API key; list teams.";
    if (t.includes("slack")) return "Slack — bot token with channels:history; list channels.";
    if (t.includes("mattermost")) return "Mattermost — base URL + PAT; list channels you belong to.";
    if (t.includes("notion")) return "Notion — integration token; search pages/databases to bind.";
    if (t.includes("confluence")) return "Confluence Cloud — base URL (/wiki) + PAT; list spaces.";
    if (t.includes("zendesk")) return "Zendesk — subdomain + email + API token; optional saved views.";
    if (t.includes("hubspot")) return "HubSpot — private app token + object kind (contacts / companies / deals).";
    if (t.includes("intercom")) return "Intercom — access token for conversation ingestion.";
    if (t.includes("fireflies")) return "Fireflies — API key for transcript listing.";
    if (t.includes("gmail") || t.includes("microsoft") || t.includes("outlook"))
      return "Paste connector_config_json from your admin runbook (OAuth connectors).";
    return "Paste connector JSON or pick another connector with a guided form.";
  }, [selectedConnector]);

  const allowedJobList = useMemo(() => JOB_OPTIONS.filter((j) => jobs[j.id]).map((j) => j.id), [jobs]);

  const readiness = useMemo(() => {
    const issues: string[] = [];
    if (!selConnector) issues.push("Choose a connector.");
    if (!displayName.trim()) issues.push("Enter a display name for this feed.");
    if (!selDomain) issues.push("Choose a domain.");
    if (!ownerId) issues.push("Assign an owner (accountable for governance).");
    if (allowedJobList.length === 0) issues.push("Select at least one allowed job.");
    if (!ingestionMode.trim()) issues.push("Choose ingestion behavior.");
    if (connectorKind === "telegram" && !botToken.trim()) issues.push("Telegram bot token is required.");
    if (connectorKind === "google_drive" && (!folderId.trim() || saJson.trim() === "{}"))
      issues.push("Google Drive: folder id and service account JSON are required.");
    if (connectorKind === "jira" && (!jiraSite.trim() || !jiraEmail.trim() || !jiraToken.trim() || !jiraProjectKey.trim()))
      issues.push("Jira: site, email, token, and project are required.");
    if (connectorKind === "trello" && (!trelloKey.trim() || !trelloToken.trim() || !trelloBoardId.trim()))
      issues.push("Trello: API key, token, and board are required.");
    if (connectorKind === "asana" && (!asanaPAT.trim() || !asanaProjectGid.trim())) issues.push("Asana: PAT and project are required.");
    if (connectorKind === "linear" && (!linearKey.trim() || !linearTeamId.trim())) issues.push("Linear: API key and team are required.");
    if (connectorKind === "slack" && (!slackBotToken.trim() || !slackChannelId.trim())) issues.push("Slack: bot token and channel are required.");
    if (connectorKind === "mattermost" && (!mmBase.trim() || !mmToken.trim() || !mmChannelId.trim()))
      issues.push("Mattermost: base URL, token, and channel are required.");
    if (connectorKind === "notion" && (!notionTok.trim() || !notionPick.trim())) issues.push("Notion: token and page/database selection are required.");
    if (connectorKind === "confluence" && (!confBase.trim() || !confAuth.trim() || !confSpaceKey.trim()))
      issues.push("Confluence: base URL, token, and space key are required.");
    if (connectorKind === "google_calendar" && (calSaJson.trim() === "{}" || !calId.trim()))
      issues.push("Google Calendar: service account JSON and calendar id are required.");
    if (connectorKind === "zendesk" && (!zdSub.trim() || !zdEmail.trim() || !zdToken.trim())) issues.push("Zendesk: subdomain, email, and API token are required.");
    if (connectorKind === "zendesk" && zdKind === "view" && !zdViewId.trim()) issues.push("Zendesk view mode: pick a saved view.");
    if (connectorKind === "hubspot" && !hsTok.trim()) issues.push("HubSpot: private app token is required.");
    if (connectorKind === "intercom" && !intercomTok.trim()) issues.push("Intercom: access token is required.");
    if (connectorKind === "fireflies" && !ffKey.trim()) issues.push("Fireflies: API key is required.");
    if (connectorKind === "oauth_blob" || connectorKind === "raw") {
      try {
        const o = JSON.parse(advancedJson) as unknown;
        if (typeof o !== "object" || o === null || Array.isArray(o)) issues.push("Advanced: connector_config_json must be a JSON object.");
      } catch {
        issues.push("Advanced: invalid JSON for connector_config_json.");
      }
    }
    const previewRecommended = createdFeedId === "";
    return { ok: issues.length === 0, issues, previewRecommended };
  }, [
    selConnector,
    displayName,
    selDomain,
    ownerId,
    allowedJobList,
    ingestionMode,
    connectorKind,
    botToken,
    folderId,
    saJson,
    createdFeedId,
    jiraSite,
    jiraEmail,
    jiraToken,
    jiraProjectKey,
    trelloKey,
    trelloToken,
    trelloBoardId,
    asanaPAT,
    asanaProjectGid,
    linearKey,
    linearTeamId,
    slackBotToken,
    slackChannelId,
    mmBase,
    mmToken,
    mmChannelId,
    notionTok,
    notionPick,
    confBase,
    confAuth,
    confSpaceKey,
    calSaJson,
    calId,
    zdSub,
    zdEmail,
    zdToken,
    zdKind,
    zdViewId,
    hsTok,
    intercomTok,
    ffKey,
    advancedJson,
  ]);

  const buildConfig = (): string => {
    if (connectorKind === "telegram") return JSON.stringify({ bot_token: botToken.trim() });
    if (connectorKind === "google_drive") {
      let sa: Json = {};
      try {
        sa = JSON.parse(saJson) as Json;
      } catch {
        sa = {};
      }
      return JSON.stringify({ folder_id: folderId.trim(), service_account: sa, max_files_per_sync: 10 });
    }
    if (connectorKind === "jira") {
      return JSON.stringify({
        jira_site_base_url: jiraSite.trim(),
        jira_email: jiraEmail.trim(),
        jira_api_token: jiraToken.trim(),
        jira_max_results: 50,
      });
    }
    if (connectorKind === "trello") return JSON.stringify({ trello_api_key: trelloKey.trim(), trello_token: trelloToken.trim() });
    if (connectorKind === "asana") return JSON.stringify({ asana_personal_access_token: asanaPAT.trim() });
    if (connectorKind === "linear") return JSON.stringify({ linear_api_key: linearKey.trim() });
    if (connectorKind === "slack") return JSON.stringify({ bot_token: slackBotToken.trim(), feed_kind: "channel" });
    if (connectorKind === "mattermost") return JSON.stringify({ mattermost_base_url: mmBase.trim(), mattermost_token: mmToken.trim() });
    if (connectorKind === "notion") {
      return JSON.stringify({ notion_integration_token: notionTok.trim(), scope: notionScope });
    }
    if (connectorKind === "confluence") {
      return JSON.stringify({
        confluence_base_url: confBase.trim(),
        confluence_auth: confAuth.trim(),
        confluence_feed_kind: "space",
      });
    }
    if (connectorKind === "google_calendar") {
      let sa: Json = {};
      try {
        sa = JSON.parse(calSaJson) as Json;
      } catch {
        sa = {};
      }
      return JSON.stringify({ service_account: sa });
    }
    if (connectorKind === "zendesk") {
      return JSON.stringify({
        zendesk_subdomain: zdSub.trim(),
        zendesk_email: zdEmail.trim(),
        zendesk_api_token: zdToken.trim(),
        zendesk_feed_kind: zdKind,
      });
    }
    if (connectorKind === "hubspot") return JSON.stringify({ hubspot_private_app_token: hsTok.trim(), hubspot_feed_kind: hsKind });
    if (connectorKind === "intercom") return JSON.stringify({ intercom_access_token: intercomTok.trim() });
    if (connectorKind === "fireflies") return JSON.stringify({ fireflies_api_key: ffKey.trim() });
    if (connectorKind === "oauth_blob" || connectorKind === "raw") return advancedJson.trim() || "{}";
    return "{}";
  };

  const externalRefForCreate = (): string => {
    switch (connectorKind) {
      case "jira":
        return jiraProjectKey.trim();
      case "trello":
        return trelloBoardId.trim();
      case "asana":
        return asanaProjectGid.trim();
      case "linear":
        return linearTeamId.trim();
      case "slack":
        return slackChannelId.trim();
      case "mattermost":
        return mmChannelId.trim();
      case "notion":
        return notionPick.trim();
      case "confluence":
        return confSpaceKey.trim();
      case "google_calendar":
        return calId.trim();
      case "zendesk":
        return zdKind === "view" ? zdViewId.trim() : "";
      default:
        return "";
    }
  };

  const goCreate = () =>
    run(async () => {
      const cfgRaw = buildConfig();
      const extRef = externalRefForCreate();
      let cfgObj: Json = {};
      try {
        cfgObj = JSON.parse(cfgRaw || "{}") as Json;
      } catch {
        throw new Error("Invalid connector_config_json");
      }
      const body = {
        connector_id: selConnector,
        domain_id: selDomain,
        owner_id: ownerId,
        display_name: displayName.trim(),
        sensitivity_level: Number(sensitivity) || 0,
        allowed_job_types_json: allowedJobList,
        ingestion_mode: ingestionMode,
        sync_mode: "manual",
        knowledge_scope: "domain_linked",
        external_ref: extRef,
        connector_config_json: cfgObj,
      };
      const created = await apiJson<{ id: string }>("/source-feeds", {
        method: "POST",
        body: JSON.stringify(body),
      });
      setCreatedFeedId(created.id);
      setFeeds(await apiJson<Json[]>("/source-feeds"));
      setStep(6);
    });

  const goPreview = () =>
    run(async () => {
      if (!createdFeedId.trim()) return;
      const out = await apiJson<Json>(`/source-feeds/${createdFeedId.trim()}/preview`, { method: "POST" });
      setPreviewSummary(JSON.stringify(out).slice(0, 1200));
    });

  const goActivate = () =>
    run(async () => {
      if (!createdFeedId.trim()) return;
      await apiJson(`/source-feeds/${createdFeedId.trim()}/activate`, { method: "POST" });
      setFeeds(await apiJson<Json[]>("/source-feeds"));
    });

  return (
    <main className="mx-auto max-w-3xl px-6 py-10">
      <Suspense fallback={null}>
        <FromCpBanner />
      </Suspense>
      <div className="mb-8 flex flex-wrap items-baseline justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Source feeds</h1>
          <p className="mt-1 text-sm text-neutral-600">
            Guided setup for a governed Source Feed · API {apiBase()} · principal {principalUserId()}
          </p>
        </div>
        <div className="flex gap-3 text-sm">
          <Link href="/" className="text-blue-700 underline">
            Home
          </Link>
          <Link href="/control-plane/sources/connectors" className="text-blue-700 underline">
            Connectors
          </Link>
        </div>
      </div>

      <DocHelpCallout slug="sources" />

      {err ? <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{err}</div> : null}

      {!feeds && !connectors ? (
        <p className="text-sm text-neutral-600">Loading…</p>
      ) : feeds && feeds.length === 0 ? (
        <div className="mb-8 rounded-lg border border-neutral-200 bg-neutral-50 px-4 py-3 text-sm text-neutral-800">
          <p className="font-medium">Connect your first Source Feed</p>
          <p className="mt-1 text-neutral-700">
            A source feed is a governed knowledge boundary. Before activation you assign ownership, domain, sensitivity, and allowed jobs so retrieval and AI
            stay controlled.
          </p>
        </div>
      ) : null}

      <div className="mb-6 flex flex-wrap gap-1 text-xs">
        {STEPS.map((label, i) => (
          <button
            key={label}
            type="button"
            className={`rounded-full border px-2 py-0.5 ${step === i ? "border-neutral-900 bg-neutral-900 text-white" : "border-neutral-200 bg-white"}`}
            onClick={() => setStep(i)}
          >
            {i + 1}. {label}
          </button>
        ))}
      </div>

      {step === 0 ? (
        <section className="space-y-3 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium">1. Choose connector</h2>
          <p className="text-xs text-neutral-600">{connectorDescription}</p>
          <div className="grid gap-2">
            {(connectors ?? []).map((c) => (
              <button
                key={asStr(c.id)}
                type="button"
                className={`rounded border px-3 py-2 text-left text-sm ${selConnector === asStr(c.id) ? "border-neutral-900 ring-1 ring-neutral-900" : "border-neutral-200"}`}
                onClick={() => setSelConnector(asStr(c.id))}
              >
                <span className="font-medium">{asStr(c.display_name)}</span>
                <span className="ml-2 text-xs text-neutral-500">{asStr(c.type)}</span>
              </button>
            ))}
          </div>
          <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50" disabled={!selConnector} onClick={() => setStep(1)}>
            Continue
          </button>
        </section>
      ) : null}

      {step === 1 ? (
        <section className="space-y-3 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium">2. Authenticate / config</h2>
          <p className="text-xs text-neutral-600">
            Connection established here does not activate the feed. The form matches the connector you chose above. Discovery calls send <code className="text-[11px]">domain_id</code> when
            you have already chosen a domain in governance; otherwise the API uses your first granted domain for the permission check.
          </p>
          <p className="text-[11px] text-neutral-500">
            Detected setup: <span className="font-mono">{connectorKind}</span>
          </p>
          {connectorKind === "telegram" ? (
            <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="bot_token" value={botToken} onChange={(e) => setBotToken(e.target.value)} />
          ) : null}
          {connectorKind === "google_drive" ? (
            <div className="space-y-2">
              <textarea className="h-28 w-full rounded border px-2 py-1 font-mono text-xs" value={saJson} onChange={(e) => setSaJson(e.target.value)} placeholder="service_account JSON" />
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                  disabled={busy || saJson.trim() === "{}"}
                  onClick={() =>
                    void run(async () => {
                      let sa: Json = {};
                      try {
                        sa = JSON.parse(saJson) as Json;
                      } catch {
                        setErr("Invalid service account JSON");
                        return;
                      }
                      const list = await apiJson<{ id: string; name: string }[]>("/integrations/google-drive/list-folders", {
                        method: "POST",
                        body: JSON.stringify({ ...domainPayload(selDomain), service_account: sa, parent_folder_id: "" }),
                      });
                      setDriveFolders(list);
                    })
                  }
                >
                  List folders at Drive root
                </button>
                {driveFolders ? <span className="text-xs text-neutral-600">{driveFolders.length} folder(s)</span> : null}
              </div>
              <label className="block text-xs font-medium text-neutral-700">
                Folder id
                <select
                  className="mt-1 w-full rounded border px-2 py-1 text-sm"
                  value={driveFolders?.some((f) => f.id === folderId) ? folderId : ""}
                  onChange={(e) => setFolderId(e.target.value)}
                  disabled={!driveFolders || driveFolders.length === 0}
                >
                  <option value="">Choose from list or type id below…</option>
                  {(driveFolders ?? []).map((f) => (
                    <option key={f.id} value={f.id}>
                      {f.name} ({f.id})
                    </option>
                  ))}
                </select>
              </label>
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="folder_id (from picker or paste)" value={folderId} onChange={(e) => setFolderId(e.target.value)} />
            </div>
          ) : null}
          {connectorKind === "jira" ? (
            <div className="space-y-3">
              <p className="text-xs text-neutral-600">
                Use an Atlassian account email and an API token from{" "}
                <a className="text-blue-700 underline" href="https://id.atlassian.com/manage-profile/security/api-tokens" target="_blank" rel="noreferrer">
                  Atlassian account → Security → API tokens
                </a>
                . Project list uses your grants: if <code className="text-[11px]">domain_id</code> is omitted, the first granted domain is used for the permission check.
              </p>
              <input
                className="w-full rounded border px-2 py-1 text-sm"
                placeholder="Site base URL (https://yourcompany.atlassian.net)"
                value={jiraSite}
                onChange={(e) => setJiraSite(e.target.value)}
              />
              <input className="w-full rounded border px-2 py-1 text-sm" type="email" placeholder="Atlassian email" value={jiraEmail} onChange={(e) => setJiraEmail(e.target.value)} />
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" type="password" placeholder="API token" value={jiraToken} onChange={(e) => setJiraToken(e.target.value)} />
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                  disabled={busy || !jiraSite.trim() || !jiraEmail.trim() || !jiraToken.trim()}
                  onClick={() =>
                    void run(async () => {
                      const payload: Record<string, string> = {
                        jira_site_base_url: jiraSite.trim(),
                        jira_email: jiraEmail.trim(),
                        jira_api_token: jiraToken.trim(),
                      };
                      if (selDomain.trim()) payload.domain_id = selDomain.trim();
                      const list = await apiJson<{ id: string; key: string; name: string }[]>("/integrations/jira/list-projects", {
                        method: "POST",
                        body: JSON.stringify(payload),
                      });
                      setJiraProjects(list);
                      if (list.length > 0 && !jiraProjectKey) setJiraProjectKey(list[0].key);
                    })
                  }
                >
                  List Jira projects
                </button>
                {jiraProjects ? <span className="text-xs text-neutral-600">{jiraProjects.length} project(s)</span> : null}
              </div>
              <label className="block text-xs font-medium text-neutral-700">
                Project
                <select
                  className="mt-1 w-full rounded border px-2 py-1 text-sm"
                  value={jiraProjectKey}
                  onChange={(e) => setJiraProjectKey(e.target.value)}
                  disabled={!jiraProjects || jiraProjects.length === 0}
                >
                  <option value="">Select after listing…</option>
                  {(jiraProjects ?? []).map((p) => (
                    <option key={p.key} value={p.key}>
                      {p.key} — {p.name}
                    </option>
                  ))}
                </select>
              </label>
            </div>
          ) : null}
          {connectorKind === "trello" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="trello_api_key" value={trelloKey} onChange={(e) => setTrelloKey(e.target.value)} />
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="trello_token" value={trelloToken} onChange={(e) => setTrelloToken(e.target.value)} />
              <button
                type="button"
                className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy || !trelloKey.trim() || !trelloToken.trim()}
                onClick={() =>
                  void run(async () => {
                    const list = await apiJson<{ id: string; name: string }[]>("/integrations/trello/list-boards", {
                      method: "POST",
                      body: JSON.stringify({ ...domainPayload(selDomain), trello_api_key: trelloKey.trim(), trello_token: trelloToken.trim() }),
                    });
                    setTrelloBoards(list);
                    if (list[0] && !trelloBoardId) setTrelloBoardId(list[0].id);
                  })
                }
              >
                List boards
              </button>
              <select className="w-full rounded border px-2 py-1 text-sm" value={trelloBoardId} onChange={(e) => setTrelloBoardId(e.target.value)} disabled={!trelloBoards?.length}>
                <option value="">Board…</option>
                {(trelloBoards ?? []).map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {connectorKind === "asana" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="asana_personal_access_token" value={asanaPAT} onChange={(e) => setAsanaPAT(e.target.value)} />
              <button
                type="button"
                className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy || !asanaPAT.trim()}
                onClick={() =>
                  void run(async () => {
                    const list = await apiJson<{ gid: string; name: string; workspace_name?: string }[]>("/integrations/asana/list-projects", {
                      method: "POST",
                      body: JSON.stringify({ ...domainPayload(selDomain), asana_personal_access_token: asanaPAT.trim() }),
                    });
                    setAsanaProjects(list);
                    if (list[0] && !asanaProjectGid) setAsanaProjectGid(list[0].gid);
                  })
                }
              >
                List projects
              </button>
              <select className="w-full rounded border px-2 py-1 text-sm" value={asanaProjectGid} onChange={(e) => setAsanaProjectGid(e.target.value)} disabled={!asanaProjects?.length}>
                <option value="">Project…</option>
                {(asanaProjects ?? []).map((p) => (
                  <option key={p.gid} value={p.gid}>
                    {(p.workspace_name ? `${p.workspace_name} / ` : "") + p.name}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {connectorKind === "linear" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="linear_api_key" value={linearKey} onChange={(e) => setLinearKey(e.target.value)} />
              <button
                type="button"
                className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy || !linearKey.trim()}
                onClick={() =>
                  void run(async () => {
                    const list = await apiJson<{ id: string; key: string; name: string }[]>("/integrations/linear/list-teams", {
                      method: "POST",
                      body: JSON.stringify({ ...domainPayload(selDomain), linear_api_key: linearKey.trim() }),
                    });
                    setLinearTeams(list);
                    if (list[0] && !linearTeamId) setLinearTeamId(list[0].id);
                  })
                }
              >
                List teams
              </button>
              <select className="w-full rounded border px-2 py-1 text-sm" value={linearTeamId} onChange={(e) => setLinearTeamId(e.target.value)} disabled={!linearTeams?.length}>
                <option value="">Team…</option>
                {(linearTeams ?? []).map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.key} — {t.name}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {connectorKind === "slack" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="Slack bot_token" value={slackBotToken} onChange={(e) => setSlackBotToken(e.target.value)} />
              <button
                type="button"
                className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy || !slackBotToken.trim()}
                onClick={() =>
                  void run(async () => {
                    const list = await apiJson<{ id: string; name: string }[]>("/integrations/slack/list-channels", {
                      method: "POST",
                      body: JSON.stringify({ ...domainPayload(selDomain), bot_token: slackBotToken.trim() }),
                    });
                    setSlackChannels(list);
                    if (list[0] && !slackChannelId) setSlackChannelId(list[0].id);
                  })
                }
              >
                List channels
              </button>
              <select className="w-full rounded border px-2 py-1 text-sm" value={slackChannelId} onChange={(e) => setSlackChannelId(e.target.value)} disabled={!slackChannels?.length}>
                <option value="">Channel…</option>
                {(slackChannels ?? []).map((ch) => (
                  <option key={ch.id} value={ch.id}>
                    #{ch.name}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {connectorKind === "mattermost" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm" placeholder="https://mattermost.example.com" value={mmBase} onChange={(e) => setMmBase(e.target.value)} />
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="personal access token" value={mmToken} onChange={(e) => setMmToken(e.target.value)} />
              <button
                type="button"
                className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy || !mmBase.trim() || !mmToken.trim()}
                onClick={() =>
                  void run(async () => {
                    const list = await apiJson<{ id: string; display_name: string; team_name?: string }[]>("/integrations/mattermost/list-channels", {
                      method: "POST",
                      body: JSON.stringify({
                        ...domainPayload(selDomain),
                        mattermost_base_url: mmBase.trim(),
                        mattermost_token: mmToken.trim(),
                      }),
                    });
                    setMmChannels(list);
                    if (list[0] && !mmChannelId) setMmChannelId(list[0].id);
                  })
                }
              >
                List channels
              </button>
              <select className="w-full rounded border px-2 py-1 text-sm" value={mmChannelId} onChange={(e) => setMmChannelId(e.target.value)} disabled={!mmChannels?.length}>
                <option value="">Channel…</option>
                {(mmChannels ?? []).map((ch) => (
                  <option key={ch.id} value={ch.id}>
                    {(ch.team_name ? `${ch.team_name} / ` : "") + (ch.display_name || ch.id)}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {connectorKind === "notion" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="notion_integration_token" value={notionTok} onChange={(e) => setNotionTok(e.target.value)} />
              <select className="w-full rounded border px-2 py-1 text-sm" value={notionScope} onChange={(e) => setNotionScope(e.target.value as "page" | "database")}>
                <option value="page">scope: page</option>
                <option value="database">scope: database</option>
              </select>
              <button
                type="button"
                className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy || !notionTok.trim()}
                onClick={() =>
                  void run(async () => {
                    const list = await apiJson<{ object: string; id: string; title: string }[]>("/integrations/notion/search", {
                      method: "POST",
                      body: JSON.stringify({ ...domainPayload(selDomain), notion_integration_token: notionTok.trim() }),
                    });
                    const filtered = list.filter((x) => x.object === notionScope);
                    setNotionItems(filtered.length ? filtered : list);
                    const first = (filtered.length ? filtered : list)[0];
                    if (first && !notionPick) setNotionPick(first.id);
                  })
                }
              >
                Search workspace
              </button>
              <select className="w-full rounded border px-2 py-1 text-sm" value={notionPick} onChange={(e) => setNotionPick(e.target.value)} disabled={!notionItems?.length}>
                <option value="">Page / database…</option>
                {(notionItems ?? []).map((n) => (
                  <option key={n.id} value={n.id}>
                    [{n.object}] {n.title || n.id}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {connectorKind === "confluence" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm" placeholder="https://x.atlassian.net/wiki" value={confBase} onChange={(e) => setConfBase(e.target.value)} />
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" type="password" placeholder="confluence_auth (PAT)" value={confAuth} onChange={(e) => setConfAuth(e.target.value)} />
              <button
                type="button"
                className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy || !confBase.trim() || !confAuth.trim()}
                onClick={() =>
                  void run(async () => {
                    const list = await apiJson<{ key: string; name: string }[]>("/integrations/confluence/list-spaces", {
                      method: "POST",
                      body: JSON.stringify({
                        ...domainPayload(selDomain),
                        confluence_base_url: confBase.trim(),
                        confluence_auth: confAuth.trim(),
                      }),
                    });
                    setConfSpaces(list);
                    if (list[0] && !confSpaceKey) setConfSpaceKey(list[0].key);
                  })
                }
              >
                List spaces
              </button>
              <select className="w-full rounded border px-2 py-1 text-sm" value={confSpaceKey} onChange={(e) => setConfSpaceKey(e.target.value)} disabled={!confSpaces?.length}>
                <option value="">Space…</option>
                {(confSpaces ?? []).map((s) => (
                  <option key={s.key} value={s.key}>
                    {s.key} — {s.name}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {connectorKind === "google_calendar" ? (
            <div className="space-y-2">
              <textarea className="h-28 w-full rounded border px-2 py-1 font-mono text-xs" value={calSaJson} onChange={(e) => setCalSaJson(e.target.value)} />
              <button
                type="button"
                className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                disabled={busy || calSaJson.trim() === "{}"}
                onClick={() =>
                  void run(async () => {
                    let sa: Json = {};
                    try {
                      sa = JSON.parse(calSaJson) as Json;
                    } catch {
                      setErr("Invalid service account JSON");
                      return;
                    }
                    const list = await apiJson<{ id: string; summary: string }[]>("/integrations/google-calendar/list-calendars", {
                      method: "POST",
                      body: JSON.stringify({ ...domainPayload(selDomain), service_account: sa }),
                    });
                    setCalList(list);
                    if (list[0] && !calId) setCalId(list[0].id);
                  })
                }
              >
                List calendars
              </button>
              <select className="w-full rounded border px-2 py-1 text-sm" value={calId} onChange={(e) => setCalId(e.target.value)} disabled={!calList?.length}>
                <option value="">Calendar…</option>
                {(calList ?? []).map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.summary} ({c.id})
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {connectorKind === "zendesk" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm" placeholder="subdomain (without .zendesk.com)" value={zdSub} onChange={(e) => setZdSub(e.target.value)} />
              <input className="w-full rounded border px-2 py-1 text-sm" type="email" placeholder="zendesk email" value={zdEmail} onChange={(e) => setZdEmail(e.target.value)} />
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" type="password" placeholder="API token" value={zdToken} onChange={(e) => setZdToken(e.target.value)} />
              <select className="w-full rounded border px-2 py-1 text-sm" value={zdKind} onChange={(e) => setZdKind(e.target.value as "all" | "view")}>
                <option value="all">All tickets</option>
                <option value="view">Saved view</option>
              </select>
              {zdKind === "view" ? (
                <>
                  <button
                    type="button"
                    className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm disabled:opacity-50"
                    disabled={busy || !zdSub.trim() || !zdEmail.trim() || !zdToken.trim()}
                    onClick={() =>
                      void run(async () => {
                        const list = await apiJson<{ id: number; title: string }[]>("/integrations/zendesk/list-views", {
                          method: "POST",
                          body: JSON.stringify({
                            ...domainPayload(selDomain),
                            zendesk_subdomain: zdSub.trim(),
                            zendesk_email: zdEmail.trim(),
                            zendesk_api_token: zdToken.trim(),
                          }),
                        });
                        setZdViews(list);
                        if (list[0] && !zdViewId) setZdViewId(String(list[0].id));
                      })
                    }
                  >
                    List views
                  </button>
                  <select className="w-full rounded border px-2 py-1 text-sm" value={zdViewId} onChange={(e) => setZdViewId(e.target.value)} disabled={!zdViews?.length}>
                    <option value="">View…</option>
                    {(zdViews ?? []).map((v) => (
                      <option key={v.id} value={String(v.id)}>
                        {v.title}
                      </option>
                    ))}
                  </select>
                </>
              ) : null}
            </div>
          ) : null}
          {connectorKind === "hubspot" ? (
            <div className="space-y-2">
              <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="hubspot_private_app_token" value={hsTok} onChange={(e) => setHsTok(e.target.value)} />
              <select className="w-full rounded border px-2 py-1 text-sm" value={hsKind} onChange={(e) => setHsKind(e.target.value as "contacts" | "companies" | "deals")}>
                <option value="contacts">contacts</option>
                <option value="companies">companies</option>
                <option value="deals">deals</option>
              </select>
            </div>
          ) : null}
          {connectorKind === "intercom" ? (
            <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="intercom_access_token" value={intercomTok} onChange={(e) => setIntercomTok(e.target.value)} />
          ) : null}
          {connectorKind === "fireflies" ? (
            <input className="w-full rounded border px-2 py-1 text-sm font-mono" placeholder="fireflies_api_key" value={ffKey} onChange={(e) => setFfKey(e.target.value)} />
          ) : null}
          {connectorKind === "oauth_blob" || connectorKind === "raw" ? (
            <div className="space-y-1">
              <p className="text-xs text-neutral-600">Paste full connector_config_json for this connector (from your runbook or OAuth setup).</p>
              <textarea className="h-40 w-full rounded border px-2 py-1 font-mono text-xs" value={advancedJson} onChange={(e) => setAdvancedJson(e.target.value)} />
            </div>
          ) : null}
          <div className="flex gap-2">
            <button type="button" className="rounded border px-3 py-1.5 text-sm" onClick={() => setStep(0)}>
              Back
            </button>
            <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white" onClick={() => setStep(2)}>
              Continue
            </button>
          </div>
        </section>
      ) : null}

      {step === 2 ? (
        <section className="space-y-3 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium">3. Map source</h2>
          <p className="text-xs text-neutral-600">Give this feed a clear name. The technical source boundary is defined by the connector config above.</p>
          <input className="w-full rounded border px-2 py-1 text-sm" placeholder="Display name (e.g. Leadership Telegram)" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
          <div className="flex gap-2">
            <button type="button" className="rounded border px-3 py-1.5 text-sm" onClick={() => setStep(1)}>
              Back
            </button>
            <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50" disabled={!displayName.trim()} onClick={() => setStep(3)}>
              Continue
            </button>
          </div>
        </section>
      ) : null}

      {step === 3 ? (
        <section className="space-y-4 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium">4. Configure governance</h2>
          <p className="text-xs text-neutral-600">These fields are required before activation. They define how this source is governed—not decorative metadata.</p>
          <div>
            <div className="mb-1 text-xs font-medium text-neutral-700">Owner</div>
            <select className="w-full rounded border px-2 py-1 text-sm" value={ownerId} onChange={(e) => setOwnerId(e.target.value)}>
              <option value="">Select user…</option>
              {(users ?? []).map((u) => (
                <option key={u.id} value={u.id}>
                  {u.name} ({u.email})
                </option>
              ))}
            </select>
            <p className="mt-1 text-[11px] text-neutral-500">The owner is responsible for the feed’s governance and operational accountability.</p>
          </div>
          <div>
            <div className="mb-1 text-xs font-medium text-neutral-700">Domain</div>
            <select className="w-full rounded border px-2 py-1 text-sm" value={selDomain} onChange={(e) => setSelDomain(e.target.value)}>
              <option value="">Select domain…</option>
              {(domains ?? []).map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
            <p className="mt-1 text-[11px] text-neutral-500">The domain affects default access, governance, and downstream routing.</p>
          </div>
          <div>
            <div className="mb-1 text-xs font-medium text-neutral-700">Sensitivity</div>
            <select className="w-full rounded border px-2 py-1 text-sm" value={sensitivity} onChange={(e) => setSensitivity(e.target.value)}>
              {SENSITIVITY_LEVELS.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label} ({s.value})
                </option>
              ))}
            </select>
            <p className="mt-1 text-[11px] text-neutral-600">{SENSITIVITY_LEVELS.find((s) => s.value === sensitivity)?.hint}</p>
          </div>
          <div>
            <div className="mb-1 text-xs font-medium text-neutral-700">Allowed jobs</div>
            <div className="space-y-1">
              {JOB_OPTIONS.map((j) => (
                <label key={j.id} className="flex items-center gap-2 text-sm">
                  <input type="checkbox" checked={!!jobs[j.id]} onChange={(e) => setJobs((prev) => ({ ...prev, [j.id]: e.target.checked }))} />
                  {j.label}
                </label>
              ))}
            </div>
          </div>
          <div className="flex gap-2">
            <button type="button" className="rounded border px-3 py-1.5 text-sm" onClick={() => setStep(2)}>
              Back
            </button>
            <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white" onClick={() => setStep(4)}>
              Continue
            </button>
          </div>
        </section>
      ) : null}

      {step === 4 ? (
        <section className="space-y-3 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium">5. Ingestion behavior</h2>
          <div className="space-y-2">
            {INGESTION_MODES.map((m) => (
              <label key={m.value} className="flex cursor-pointer gap-2 rounded border border-neutral-200 p-2 text-sm">
                <input type="radio" name="ingestion" checked={ingestionMode === m.value} onChange={() => setIngestionMode(m.value)} />
                <span>
                  <span className="font-medium">{m.label}</span>
                  <span className="mt-0.5 block text-xs text-neutral-600">{m.hint}</span>
                </span>
              </label>
            ))}
          </div>
          <p className="text-xs text-neutral-500">Sync mode is incremental/event-driven per connector implementation; only one mode may be meaningful for a given connector.</p>
          <div className="flex gap-2">
            <button type="button" className="rounded border px-3 py-1.5 text-sm" onClick={() => setStep(3)}>
              Back
            </button>
            <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white" onClick={() => setStep(5)}>
              Continue
            </button>
          </div>
        </section>
      ) : null}

      {step === 5 ? (
        <section className="space-y-3 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium">6. Review & readiness</h2>
          <ul className="list-disc space-y-1 pl-5 text-sm text-neutral-800">
            <li>Connector: {asStr(selectedConnector?.display_name) || "—"}</li>
            <li>Wizard path: {connectorKind}</li>
            <li>External ref: {externalRefForCreate() || "—"}</li>
            <li>Display name: {displayName || "—"}</li>
            <li>Domain: {(domains ?? []).find((d) => d.id === selDomain)?.name ?? (selDomain || "—")}</li>
            <li>Owner: {(users ?? []).find((u) => u.id === ownerId)?.name ?? (ownerId || "—")}</li>
            <li>Sensitivity: {SENSITIVITY_LEVELS.find((s) => s.value === sensitivity)?.label ?? sensitivity}</li>
            <li>Allowed jobs: {allowedJobList.join(", ") || "—"}</li>
            <li>Ingestion: {ingestionMode}</li>
          </ul>
          <div className={`rounded-md border px-3 py-2 text-sm ${readiness.ok ? "border-green-200 bg-green-50 text-green-900" : "border-amber-200 bg-amber-50 text-amber-950"}`}>
            {readiness.ok ? <p className="font-medium">Ready to create draft feed</p> : <p className="font-medium">Not ready yet</p>}
            {readiness.issues.length > 0 ? (
              <ul className="mt-2 list-disc pl-5">
                {readiness.issues.map((i) => (
                  <li key={i}>{i}</li>
                ))}
              </ul>
            ) : null}
            {readiness.previewRecommended && createdFeedId === "" ? (
              <p className="mt-2 text-xs">After creation, run preview before activation (recommended).</p>
            ) : null}
          </div>
          <div className="flex gap-2">
            <button type="button" className="rounded border px-3 py-1.5 text-sm" onClick={() => setStep(4)}>
              Back
            </button>
            <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50" disabled={!readiness.ok || busy} onClick={() => void goCreate()}>
              Create draft feed
            </button>
          </div>
        </section>
      ) : null}

      {step === 6 ? (
        <section className="space-y-3 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium">7. Create & preview</h2>
          <p className="text-xs text-neutral-600">
            Draft id: <code className="rounded bg-neutral-100 px-1">{createdFeedId || "—"}</code>
          </p>
          {!createdFeedId ? (
            <p className="text-sm text-amber-800">No draft yet—go back to review and choose &quot;Create draft feed&quot;, or create again from step 6.</p>
          ) : null}
          {previewSummary ? (
            <pre className="max-h-48 overflow-auto rounded border bg-neutral-50 p-2 text-[11px] text-neutral-800">{previewSummary}</pre>
          ) : (
            <p className="text-xs text-neutral-600">Preview loads a bounded sample from the connector without activating canonical publication.</p>
          )}
          <div className="flex flex-wrap gap-2">
            <button type="button" className="rounded border px-3 py-1.5 text-sm" onClick={() => setStep(5)}>
              Back
            </button>
            <button type="button" className="rounded-md bg-neutral-800 px-3 py-1.5 text-sm text-white disabled:opacity-50" disabled={!createdFeedId || busy} onClick={() => goPreview()}>
              Run preview
            </button>
            <Link href={createdFeedId ? `/control-plane/sources/feeds/${encodeURIComponent(createdFeedId)}` : "#"} className={`rounded-md px-3 py-1.5 text-sm ${createdFeedId ? "bg-neutral-100 underline" : "pointer-events-none opacity-40"}`}>
              Open feed detail
            </Link>
            <button type="button" className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white" onClick={() => setStep(7)}>
              Continue to activate
            </button>
          </div>
        </section>
      ) : null}

      {step === 7 ? (
        <section className="space-y-3 rounded-lg border border-neutral-200 p-4">
          <h2 className="text-sm font-medium">8. Activate</h2>
          <p className="text-xs text-neutral-600">Activation moves the feed to an operational state. Run preview first if you have not already.</p>
          <button
            type="button"
            className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm text-white disabled:opacity-50"
            disabled={!createdFeedId || busy || !readiness.ok}
            onClick={() => goActivate()}
          >
            Activate feed
          </button>
          <div className="flex flex-wrap gap-2 text-sm">
            <button type="button" className="rounded border px-3 py-1.5" onClick={() => setStep(6)}>
              Back
            </button>
            <button
              type="button"
              className="rounded border px-3 py-1.5"
              disabled={busy}
              onClick={() => run(async () => setFeeds(await apiJson<Json[]>("/source-feeds")))}
            >
              Refresh feed list
            </button>
          </div>
        </section>
      ) : null}

      {feeds && feeds.length > 0 ? (
        <section className="mt-10">
          <h2 className="text-sm font-medium">Existing feeds</h2>
          <ul className="mt-2 space-y-1 text-xs font-mono text-neutral-700">
            {feeds.map((f) => (
              <li key={asStr(f.id)}>
                {asStr(f.display_name)} · {asStr(f.status)} · {asStr(f.id)}{" "}
                <Link href={`/control-plane/sources/feeds/${encodeURIComponent(asStr(f.id))}`} className="text-blue-700 underline">
                  detail
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </main>
  );
}

// Wrap in Suspense because SourceFeedsPageInner calls useSearchParams() directly,
// which Next.js 15 prerender requires under a Suspense boundary.
export default function SourceFeedsPage() {
  return (
    <Suspense fallback={<main className="p-10 text-sm text-neutral-600">Loading source feeds…</main>}>
      <SourceFeedsPageInner />
    </Suspense>
  );
}
