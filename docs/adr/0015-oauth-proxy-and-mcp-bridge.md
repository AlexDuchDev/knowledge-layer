# ADR-0015: OAuth 2.1 proxy and MCP bridge

## Status

**Accepted (2026-04-30).** Initial scope: OAuth proxy alone (v0.5.0). Promoted to "OAuth + MCP" when the v0.5.1 MCP endpoint lands and consumes the proxy's bearers.

## Context

External AI clients — Claude Desktop, Cursor, IDE plugins — use the **Model Context Protocol** (MCP) to consume tools over HTTP. To let these clients query a Knowledge Layer instance, an operator needs to (a) authenticate the human at the keyboard, (b) bind the resulting MCP session to a real KL principal so `AccessEvaluator` decides every tool call.

A KL operator already has an OIDC issuer (Keycloak / Auth0 / Okta / Dex). What they lack is a way to bridge an OIDC identity into a KL bearer that an MCP client can hold. The two natural approaches are:

1. **Make MCP clients speak directly to the operator's IDP.** Each MCP client would have to support the operator's specific dynamic-client registration, redirect URLs, and scope conventions. Today most MCP clients support only the **OAuth 2.1 dynamic-registration + PKCE** flow as defined by the [MCP authorization spec](https://modelcontextprotocol.io/specification/draft/basic/authorization).
2. **Make Knowledge Layer itself look like an OAuth 2.1 authorization server**, fronting whichever OIDC issuer the operator runs. This is what Hugr does (`pkg/auth` with `MCP_ENABLED=true` flips on `/oauth/*` proxy endpoints).

We adopt option 2.

## Decision

Knowledge Layer ships a **stateless OAuth 2.1 proxy** at `apps/api/internal/oauth_proxy/`. Endpoints:

- `GET /.well-known/oauth-authorization-server` — RFC 8414 metadata; tells MCP clients where the proxy's other endpoints live.
- `GET /oauth/authorize` — entry point; redirects the user-agent to the operator's IDP after recording the MCP client's redirect URI and PKCE challenge in a signed `state` payload.
- `GET /oauth/callback` — IDP returns here. The proxy verifies the state HMAC, exchanges the IDP code for an `id_token`, maps the verified `sub` claim to a KL user via `TokenBridge`, and redirects the MCP client back to its registered URI with our own one-time auth code.
- `POST /oauth/token` — MCP client exchanges the auth code (plus PKCE `code_verifier`, plus optional `client_secret` for confidential clients) for a JWT bearer. The JWT's `sub` claim is the KL principal UUID; subsequent `/mcp` requests (v0.5.1) carry it.
- `POST /oauth/register` — RFC 7591 dynamic-client registration. MCP clients self-register, receive a `client_id` and (optionally) a one-shot `client_secret`. Stored in the `oauth_clients` table.

**Single-tenant constraint preserved**. ADR-0014 stays in force: no tenant claim in tokens, no per-tenant routing in the proxy. One Knowledge Layer instance = one organization.

**AccessEvaluator-mandatory contract**. Every MCP tool call (when v0.5.1 lands) MUST pass through `AccessEvaluator.Evaluate(...)` exactly like the REST API surface. The OAuth bearer maps to a real `users.id`; nothing on the MCP path bypasses the 9-step evaluator. There is no admin-bypass token. A bearer whose `sub` doesn't resolve to a known user is rejected with **401**, not 403 — the user has no Knowledge Layer identity at all.

**State is stateless**. The `state` parameter shipped to the IDP is a base64url-encoded JSON payload signed with `OAUTH_SECRET_KEY` (HMAC-SHA256, ≥32 bytes). On callback we verify the HMAC and reject anything whose age exceeds 10 minutes. No DB write per authorization request.

**Auth codes are short-lived in-process**. The one-time code we hand the MCP client between callback and token exchange lives 2 minutes in `sync.Map`. PKCE makes single-instance memory safe; multi-pod deployments need either sticky LB or a Redis backend (out of scope per ADR-0014's single-instance stance).

**JWT signing is HMAC-SHA256** with `OAUTH_SECRET_KEY`. Symmetric so we can verify without a KMS round-trip. Token lifetime is 1 hour; refresh is not supported in v0.5.0 — clients re-authorize. Acceptable trade-off because the IDP-side session typically extends well beyond an hour and `/oauth/authorize` flows through silently when the IDP cookie is fresh.

## Required env (production)

Validated in `config.ValidateAPI` when `OAUTH_PROXY_ENABLED=true`:

- `OAUTH_SECRET_KEY` ≥ 32 bytes (signs state and JWTs).
- `OIDC_ISSUER_URL` — operator's IDP discovery URL.
- `OIDC_CLIENT_ID` — proxy's client ID at the IDP.
- `OIDC_CLIENT_SECRET` — required in production. Local/staging may run public IDP for testing.
- `OIDC_SUB_COLUMN` — `email` (default) or `external_idp_subject`. Determines how the OIDC `sub` claim maps to a `users` row.

Optional:

- `OAUTH_PROXY_ISSUER` — explicit issuer URL for `.well-known` (defaults to `APP_PUBLIC_URL`).
- `OAUTH_PROXY_CALLBACK` — explicit callback URL (defaults to `${OAUTH_PROXY_ISSUER}/oauth/callback`).

## Consequences

### Positive

- Operators already running OIDC drop in their issuer URL and the proxy works without code changes on either side.
- MCP clients (Claude Desktop, Cursor) use a single, public, RFC-compliant flow.
- Every issued bearer maps to a real KL user; AccessEvaluator decides every tool call.
- State encryption + signed JWTs mean we don't need a session store.

### Negative

- Single-instance assumption persists. Multi-pod operators need sticky LB or a code-backend swap.
- HMAC-symmetric JWTs mean every API replica needs the secret key. Acceptable per the single-instance stance.
- Refresh tokens not supported; clients re-authorize hourly. UX cost is small (silent IDP redirect when IDP session is fresh).

### Neutral

- The proxy is opt-in (`OAUTH_PROXY_ENABLED=false` default). Operators who don't want MCP keep zero new attack surface.

## Alternatives considered

- **Reuse `connectoroauth/`** (the existing per-connector OAuth glue). Rejected: that package is built for *outbound* connector auth (Gmail / M365). Mixing inbound proxy + outbound client into one package would couple the two lifecycles awkwardly. v0.6.0's `openapi_v3` connector may want some of `connectoroauth`'s primitives; that's where reuse is considered, not here.
- **External Dex / oauth2-proxy in front of the API**. Operator-side complexity grows (two stateful services). Doesn't solve the "bearer → KL principal" mapping; AccessEvaluator wiring still needs to live in-process. We deferred this option as a deployment topology operators can choose to add but not require.
- **Skip OAuth and use a static API token from KL's existing PAT**. Some MCP clients accept it; many don't. The MCP authorization spec specifically calls out OAuth 2.1 + PKCE.

## Documentation updates

- README → add operator note about MCP/OAuth opt-in (v0.5.1 when MCP ships).
- `docs/PRODUCTION_HARDENING.md` → § for OAuth proxy hardening (key rotation cadence, redirect_uri allow-list discipline).
- `docs/SECRET_ROTATION.md` → procedure for rotating `OAUTH_SECRET_KEY` (in-flight authorizations break; document the trade-off).
- `docs/CONFIG_ENV.md` → "OAuth proxy" section.

## Revisiting

- Multi-instance MCP fan-out → revisit if operators ask for HA MCP. Likely path: move authCodes to Redis, key off the same HMAC.
- Token refresh → revisit if user complaints about hourly re-auth become noisy. Likely path: introduce an opaque refresh token backed by `oauth_refresh_tokens` table, keep JWTs short-lived.
- Asymmetric JWTs (RS256/ES256) → revisit if a downstream service needs to verify our bearers without holding `OAUTH_SECRET_KEY`. Today there is no such consumer.
