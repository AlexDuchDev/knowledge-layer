# MCP endpoint — operator guide

This document covers operating the **Model Context Protocol** endpoint shipped in [v0.5.1](../../CHANGELOG.md). It lets external AI clients (Claude Desktop, Cursor, IDE plugins) consume Knowledge Layer tools after authenticating through the [v0.5.0 OAuth proxy](../adr/0015-oauth-proxy-and-mcp-bridge.md).

> **Single-tenant constraint preserved.** ADR-0014 stays in force. Every issued bearer maps to a real `users.id` row; there is no admin bypass. Tool calls go through the same 9-step `AccessEvaluator` as REST.

---

## Architecture at a glance

```
┌──────────────────┐                ┌─────────────────────────────────┐
│ Claude Desktop   │                │ Knowledge Layer API             │
│ (or Cursor /     │                │  ┌──────────────────────────┐   │
│  IDE plugin)     │                │  │ /.well-known/oauth-      │   │
│                  │                │  │  authorization-server    │   │
│  ┌────────────┐  │   discover     │  └──────────────────────────┘   │
│  │ MCP client │──┼───────────────▶│  /oauth/register             ◀──┘ dynamic
│  │            │  │   register     │  /oauth/authorize  ───┐         registration
│  │            │  │                │  /oauth/token         │
│  └─────┬──────┘  │                └────────┬───────┬──────┼────────┘
│        │         │                         │       │      │
│        │         │                         ▼       │      ▼
│   redirect to    │                    ┌─────────┐  │  ┌──────────┐
│   IDP via 302    │                    │ TokenBr.│  │  │ AccessEv │
│        ▼         │                    │ sub→KL  │  │  │ 9 steps  │
│  ┌────────────┐  │  3-leg OAuth+PKCE  │ users.id│  │  └──────────┘
│  │ User-agent │──┼───────────────────▶└─────────┘  │       ▲
│  │ + IDP      │  │                                 │       │
│  └────────────┘  │  bearer JWT (HS256)             │       │
│        │         │◀────────────────────────────────┘       │
│        │ POST    │                                         │
│        └─/mcp ─▶ │  withAccessGuard → tool handler ────────┘
└──────────────────┘
```

The key flow:

1. MCP client discovers Knowledge Layer via `/.well-known/oauth-authorization-server` (RFC 8414).
2. Client self-registers via `POST /oauth/register` (RFC 7591). Operator never touches this — it's automatic.
3. Client kicks off PKCE auth-code flow: 302 to operator's OIDC issuer → user logs in → callback to `/oauth/callback` → KL mints a one-time code → redirects back to the client.
4. Client exchanges the code at `POST /oauth/token` for a 1-hour HS256 JWT.
5. Client calls `POST /mcp` with `Authorization: Bearer <jwt>`. Bearer middleware verifies signature, extracts the principal UUID, calls `AccessEvaluator.Evaluate` per tool. Allow → tool runs. Deny → MCP-shaped error.

---

## Prerequisites

| Requirement | Why |
|---|---|
| OIDC issuer reachable from the API container | OAuth proxy performs discovery at startup; failure aborts the proxy |
| OIDC client registered in your IDP for KL | KL acts as a confidential client at the IDP |
| `OAUTH_PROXY_ENABLED=true` and required env (see below) | Hardening rejects `MCP_ENABLED=true` without `OAUTH_PROXY_ENABLED=true` |
| Users known to KL (matching IDP `sub` or email) | `TokenBridge` rejects bearers whose `sub` doesn't resolve to a `users` row (401) |
| `APP_PUBLIC_URL` is the proxy's `.well-known` issuer | Set explicitly or override via `OAUTH_PROXY_ISSUER` |

---

## Enabling the endpoint

Set in your `.env` or container config:

```bash
# Required: OAuth proxy must be on first
OAUTH_PROXY_ENABLED=true
OAUTH_SECRET_KEY=<32-byte hex; rotate via docs/SECRET_ROTATION.md>
OIDC_ISSUER_URL=https://idp.example.com/realms/main
OIDC_CLIENT_ID=knowledge-layer
OIDC_CLIENT_SECRET=<your secret>           # required in production
OIDC_SUB_COLUMN=email                      # or 'external_idp_subject'

# Optional override (defaults shown)
# OAUTH_PROXY_ISSUER=https://kl.example.com
# OAUTH_PROXY_CALLBACK=https://kl.example.com/oauth/callback

# Then: enable MCP
MCP_ENABLED=true
```

Restart the API. `config.ValidateAPI` checks the combination at startup; bad config crashes loud rather than serving a half-broken endpoint.

Verify the endpoints respond:

```bash
curl -s https://kl.example.com/.well-known/oauth-authorization-server | jq '.issuer'
# → "https://kl.example.com"

curl -i https://kl.example.com/mcp
# → 401 Unauthorized + WWW-Authenticate: Bearer realm="mcp", error="invalid_token"
```

---

## Wiring Claude Desktop

In Claude Desktop's developer settings, add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "knowledge-layer": {
      "url": "https://kl.example.com/mcp"
    }
  }
}
```

Restart Claude Desktop. On first use it will:
1. Read `.well-known` to find OAuth endpoints.
2. Register itself via `POST /oauth/register`.
3. Open a browser tab to your IDP for login.
4. Receive a JWT, cache it locally, start using the tools.

The user logs in once per IDP-session lifetime (typically 8–24 h). KL's bearer expires after 1 h; on refresh Claude Desktop re-runs the auth-code flow silently (the IDP cookie is still fresh, so no user interaction).

---

## Wiring Cursor / IDE plugins

Cursor (and most IDE-plugin MCP clients) accept the same shape:

```json
{
  "mcp": {
    "servers": {
      "knowledge-layer": "https://kl.example.com/mcp"
    }
  }
}
```

Refer to your client's documentation for the exact key path.

---

## Initial tool set (v0.5.1)

| Tool | What it does | Input schema | Wraps |
|---|---|---|---|
| `kl_search` | Keyword search across permitted entities | `q` (required), `type`, `domain_id`, `limit` | `Retrieval.SearchScoped` |
| `kl_ask_global` | Synthesized Q&A through the privacy gateway | `question` (required), `include_related` | `Retrieval.AskGlobal` |
| `kl_get_entity` | Fetch a single entity by UUID | `entity_id` (required) | `Entities.Get` |

Every tool is wrapped in `withAccessGuard`. The `TestNew_allToolsAccessGuarded` static-contract test prevents regression: any new tool added without the wrapper fails CI.

Adding a tool? See [`apps/api/internal/mcp/server.go`](../../apps/api/internal/mcp/server.go) — pattern is `newGuardedTool(name, desc, action, resourceType, schema, handler)` then append to the slice in `New(deps)`.

---

## Operating concerns

### Token lifetime

JWTs live 1 h (`signJWT`). No refresh tokens in v0.5.1. MCP clients silently re-authorize; users notice only if their IDP session has expired.

To **force-revoke all in-flight bearers**, rotate `OAUTH_SECRET_KEY`. See [SECRET_ROTATION.md](../SECRET_ROTATION.md).

### Per-client revocation

To block one MCP client without rotating the global key:

```sql
UPDATE oauth_clients SET revoked_at = now() WHERE client_id = '<client_id>';
```

In-flight bearers for that client expire naturally within 1 h. `/oauth/token` rejects further attempts immediately.

### Audit

`oauth.client.registered` and `oauth.token.issued` log to **stderr** in v0.5.1. Promotion to `audit_events` rows is a v0.5.x follow-up — once operators ask to query MCP traffic via `/audit-events`. For now, parse stderr / container logs:

```bash
docker compose logs api | grep -E 'oauth\.(client|token)\.'
```

Tool-call audit (`mcp.tool.invoked`) is the same story: stderr today, audit_events later.

### Single-instance assumption

Auth codes live 2 minutes in `sync.Map` per [ADR-0015](../adr/0015-oauth-proxy-and-mcp-bridge.md). Multi-pod operators need:
- Sticky load-balancing on `/oauth/*` so the callback hits the same pod that issued the auth code, **or**
- Wait for the v0.5.x backend swap to Redis (no current operator demand).

Per ADR-0014, single-instance is the documented stance.

### Memory / connection footprint

MCP traffic uses the existing API's pool. No new DB connections. Each MCP request is bounded by the access-guard's single `AccessEvaluator.Evaluate` call + the underlying tool's own DB queries.

### Metrics

`/metrics` already records `http_requests_total{route="/mcp"}` because the standard Prometheus middleware sees every Fiber route. No MCP-specific counters in v0.5.1.

---

## Common operator questions

**Q: Why does my MCP client get 401 even after a successful login?**
The IDP authenticated the user but `TokenBridge` couldn't find a matching `users.id`. Either:
- `OIDC_SUB_COLUMN=email` and the IDP-issued email doesn't match a row in `users.email`, or
- `OIDC_SUB_COLUMN=external_idp_subject` and you haven't populated that column.

Provision the user in KL first, then retry. There is no auto-provisioning in v0.5.1 — explicit by design.

**Q: Can I let multiple operators each have their own MCP client?**
Yes — each MCP client self-registers with its own `client_id`. They share the same OAuth proxy and same access semantics; the principal that ends up in the JWT is whoever logged in.

**Q: Can I disable just one tool without unmounting `/mcp`?**
Not in v0.5.1. The 3 initial tools are registered unconditionally. v0.5.x will likely add a `MCP_ALLOW_TOOLS=kl_search,kl_get_entity` filter — file an issue if you need it.

**Q: Does MCP traffic count toward the existing `/ops/health` worker tracker?**
No. `/mcp` is on the API process; worker health endpoints are on `cmd/jobworker` and `cmd/connectorworker`, separate process group.

**Q: Can my MCP client write data via the protocol?**
Not yet. v0.5.1 ships read-only tools (`kl_search`, `kl_ask_global`, `kl_get_entity`). Mutation tools (`kl_publish_entity`, `kl_patch_entity`, etc.) come later as the access-guard pattern proves itself in production.

---

## Related docs

- [ADR-0015](../adr/0015-oauth-proxy-and-mcp-bridge.md) — design of the OAuth proxy + MCP bridge
- [ADR-0014](../adr/0014-single-tenant-deployment-stance.md) — single-tenant invariant
- [`docs/CONFIG_ENV.md`](../CONFIG_ENV.md) — `MCP_*` and `OAUTH_*` / `OIDC_*` env table
- [`docs/SECRET_ROTATION.md`](../SECRET_ROTATION.md) — rotating `OAUTH_SECRET_KEY`
- [`docs/PRODUCTION_HARDENING.md`](../PRODUCTION_HARDENING.md) — production checklist
