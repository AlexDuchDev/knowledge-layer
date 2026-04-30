-- OAuth dynamic-registration table for the v0.5.0 OAuth 2.1 proxy. Each row
-- is one MCP client (Claude Desktop, Cursor, etc.) that registered itself
-- via /oauth/register per RFC 7591. State is intentionally minimal — most
-- of the OAuth flow stays stateless via signed `state` payloads (see
-- oauth_proxy/state.go); this table only persists what dynamic-registration
-- demands: a client_id, a verifiable secret, and the redirect_uri allow-list
-- the callback handler enforces.
--
-- Visibility: this table is admin-only. There is no public API for listing
-- registered clients today; operators inspect via psql or the AuditOps reads
-- of `oauth.client.registered` events.
CREATE TABLE oauth_clients (
    client_id TEXT PRIMARY KEY,
    -- bcrypt of the secret returned to the client at registration time.
    -- We never store the plaintext secret. Empty string for "public clients"
    -- (PKCE-only, no secret) — those still need redirect_uri verification.
    client_secret_hash TEXT NOT NULL DEFAULT '',
    -- Allow-list. Callbacks whose redirect_uri does not match any value here
    -- are rejected before forwarding the IDP code.
    redirect_uris TEXT[] NOT NULL DEFAULT '{}',
    -- Display name supplied by the client for operator-side identification.
    -- Untrusted; never rendered as HTML.
    client_name TEXT NOT NULL DEFAULT '',
    -- Token endpoint auth method. "none" for public PKCE clients,
    -- "client_secret_basic" / "client_secret_post" for confidential.
    token_endpoint_auth_method TEXT NOT NULL DEFAULT 'none',
    -- Operator may revoke via UPDATE ... SET revoked_at = now().
    -- Active flow paths must check this column.
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_oauth_clients_active ON oauth_clients (created_at DESC) WHERE revoked_at IS NULL;
