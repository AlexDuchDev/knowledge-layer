# oauth_proxy

Stateless OAuth 2.1 authorization-server proxy fronting an operator-supplied OIDC issuer (Keycloak / Auth0 / Okta / Dex). Shipped in v0.5.0; consumed by the v0.5.1 MCP endpoint.

Design rationale + alternatives in [ADR-0015](../../../../docs/adr/0015-oauth-proxy-and-mcp-bridge.md). Operator setup in [docs/operations/mcp.md](../../../../docs/operations/mcp.md).

## Surface

```go
type Server struct {
    IDP *IDPClient; Clients *ClientRepo; Bridge *TokenBridge
    SecretKey []byte; Issuer, RedirectURL string
}
func (s *Server) Mount(r fiber.Router)
func (s *Server) VerifyBearer(jwt string) (uuid.UUID, error)  // used by mcp/route.go
func PublicPaths() []string                                    // for routing allow-list
```

## Files

- `state.go` — HMAC-SHA256 signed `state` payload (OAuth `state` query param). 6 unit tests cover round-trip, tamper, wrong key, expiry, short-key rejection, malformed input.
- `idp_client.go` — `coreos/go-oidc/v3` wrapper. OIDC discovery at startup; `AuthorizeURL` builds the redirect, `Exchange` swaps IDP code for `id_token` and verifies it.
- `token_bridge.go` — `Resolve(ctx, claims)` maps OIDC `sub` to a KL `users.id`. Default `OIDC_SUB_COLUMN=email` matches against `users.email`. No auto-provision; absent user → 401 (not 403).
- `clients.go` — `ClientRepo` over the `oauth_clients` table (migration 000043). RFC 7591 `Register` returns a one-shot `client_secret` for confidential clients (bcrypt-hashed at rest); PKCE-only "public" clients accept `auth_method=none`. `AllowsRedirect` is verbatim string match (no path-prefix loosening).
- `server.go` — five Fiber handlers: `metadata`, `register`, `authorize`, `callback`, `token`. PKCE-S256 verified at token endpoint; auth codes live 2 minutes in `sync.Map`.

## Endpoints

| Path | Method | Auth | Purpose |
|---|---|---|---|
| `/.well-known/oauth-authorization-server` | GET | none | RFC 8414 discovery doc |
| `/oauth/register` | POST | none | RFC 7591 dynamic-client registration |
| `/oauth/authorize` | GET | none | Entry point; signs state + 302 to IDP |
| `/oauth/callback` | GET | none | IDP returns here; exchanges code, mints KL auth code |
| `/oauth/token` | POST | client_secret_basic / none + PKCE | Returns 1-hour HS256 JWT bearer |

## Statelessness

- The `state` query param is HMAC-signed with `OAUTH_SECRET_KEY` (≥32 bytes). No DB write per authorization request.
- KL's auth codes (handed back to the MCP client between `/oauth/callback` and `/oauth/token`) live 2 minutes in `sync.Map`. PKCE-S256 verifier check happens at token exchange.
- Issued JWTs are HS256 signed with the same `OAUTH_SECRET_KEY`. No KMS round-trip on verify; every API replica with the same secret can verify.

## Single-instance assumption

Per ADR-0014, KL is one-instance-per-org. Multi-pod deployments need sticky LB on `/oauth/*` so the callback hits the same pod that created the auth code. v0.5.x has no Redis backend for codes; if multi-pod is required, the swap is straightforward (typed `LoadAndDelete` over Redis with a 2-minute TTL).

## Token revocation

| Scope | How |
|---|---|
| Per-client | `UPDATE oauth_clients SET revoked_at = now() WHERE client_id = ...`. In-flight bearers expire in ≤1 h. |
| Global | Rotate `OAUTH_SECRET_KEY`. Every JWT becomes invalid immediately. See [SECRET_ROTATION.md §9](../../../../docs/SECRET_ROTATION.md). |

## Audit

`oauth.client.registered` and `oauth.token.issued` log to **stderr** in v0.5.x. Promotion to `audit_events` rows is a v0.5.x patch follow-up — it lands when operator demand for `/audit-events` queries on MCP traffic surfaces.

## Adding a new endpoint to the unauth allow-list

Edit `Server.Mount` and add the path to the public-by-design set. RFC 8414 / 7591 endpoints are public by spec; everything else MUST go through `principalMiddleware` or the v0.5.1 MCP bearer middleware.
