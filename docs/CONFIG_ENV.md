# Configuration (environment)

Authoritative template: [.env.example](/.env.example) in the repository root.

This file replaces RTF-wrapped “Config and environments” exports for day-to-day use.

## API

| Variable | Description |
|----------|-------------|
| `APP_ENV` | Runtime environment name |
| `API_PORT` | HTTP port |
| `DATABASE_URL` | PostgreSQL DSN |
| `OPENSEARCH_URL` | OpenSearch base URL or empty |
| `AUTH_MODE` | `development_header` or `session` |
| `SESSION_SECRET` | HMAC secret for session cookies (session mode) |
| `ALLOW_SELF_REGISTRATION` | `true`/`false` — default false |
| `APP_PUBLIC_URL` | Public API URL for links in emails |
| `AUTO_BOOTSTRAP_INSTANCE` | Local: default **on** (set `0`/`false`/`no` to disable). Staging/production: **off** unless `1`/`true`. When on and the DB has **no domains**, API startup creates the first admin + domain (same as `POST /instance/bootstrap`). |
| `BOOTSTRAP_ADMIN_EMAIL` | Required for non-local auto-bootstrap; local default `admin@local.test`. |
| `BOOTSTRAP_ADMIN_PASSWORD` | Min 8 characters; local default `changeme12345` (change immediately if exposed). |
| `BOOTSTRAP_ADMIN_NAME` | Display name; default `Administrator`. |
| `BOOTSTRAP_DOMAIN_NAME` | First workspace name; default `Default`. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` | Outbound email |
| `BUILD_VERSION` | Optional version string for `/settings/instance` |
| `BLOBSTORE_BACKEND` | Empty or `nop` (default): discard blob bytes but return a URI; `s3`: S3-compatible uploads (MinIO / R2 / AWS). |
| `BLOBSTORE_S3_ENDPOINT` | Host:port for S3 API (no `https://` prefix), e.g. `localhost:9000` or `xxx.r2.cloudflarestorage.com`. |
| `BLOBSTORE_S3_REGION` | Region string (default `us-east-1`). |
| `BLOBSTORE_S3_BUCKET` | Target bucket name. |
| `BLOBSTORE_S3_ACCESS_KEY`, `BLOBSTORE_S3_SECRET_KEY` | API credentials. |
| `BLOBSTORE_S3_USE_SSL` | `true`/`false` — default `true`. |
| `BLOBSTORE_S3_PATH_STYLE` | `true` for many MinIO setups (path-style addressing). |

## OAuth connector onboarding (API)

These variables enable the server-side OAuth flow for Gmail and Microsoft 365. When set, the web wizard can request an authorize URL, the provider redirects back to the API callback, and the API issues a one-time `oauth_sid` handoff to the web.

| Variable | Description |
|----------|-------------|
| `API_PUBLIC_URL` | Browser-reachable API base for OAuth `redirect_uri` (default: `APP_PUBLIC_URL`). Example: `https://api.example.com`. |
| `OAUTH_WEB_REDIRECT_URL` | Web URL to redirect to after OAuth callback; `oauth_sid` is appended as a query param. Default: `http://localhost:3000/source-feeds`. |
| `GMAIL_OAUTH_CLIENT_ID` | Google Cloud OAuth client id for Gmail delegated OAuth. |
| `GMAIL_OAUTH_CLIENT_SECRET` | Google Cloud OAuth client secret for Gmail delegated OAuth. |
| `MICROSOFT_OAUTH_CLIENT_ID` | Azure app registration client id. |
| `MICROSOFT_OAUTH_CLIENT_SECRET` | Azure app registration client secret. |
| `MICROSOFT_OAUTH_TENANT` | Tenant string for Microsoft OAuth endpoints (default `common`). |

**Operator check (S3):** After configuring `BLOBSTORE_BACKEND=s3`, verify connectivity with a controlled upload in staging (or MinIO against the same endpoint) before relying on blob retention in production. Automated MinIO tests are optional (`MINIO_INTEGRATION` not wired in CI by default).

## OAuth 2.1 proxy (API, v0.5.0)

Stateless OAuth proxy fronting an operator-supplied OIDC issuer. Off by default. Enables RFC 8414 metadata + RFC 7591 dynamic registration + auth-code + PKCE so MCP clients (Claude Desktop, Cursor — when v0.5.1 ships) can authenticate against the operator's IDP. Every issued bearer maps to a real `users` row; `AccessEvaluator` decides every downstream call. See [ADR-0015](adr/0015-oauth-proxy-and-mcp-bridge.md).

| Variable | Description |
|----------|-------------|
| `OAUTH_PROXY_ENABLED` | `true` to mount `/oauth/*` and `/.well-known/oauth-authorization-server` (default `false`). |
| `OAUTH_SECRET_KEY` | ≥32 bytes. Signs OAuth state HMAC + issued JWT bearers. Rotate via [SECRET_ROTATION.md](SECRET_ROTATION.md) (in-flight authorizations break — acceptable trade-off). |
| `OIDC_ISSUER_URL` | Operator's IDP discovery URL. Reachable at API startup. |
| `OIDC_CLIENT_ID` | The proxy's client_id at the IDP. |
| `OIDC_CLIENT_SECRET` | Required in production. Local/staging may run public-client IDP for testing. |
| `OIDC_SUB_COLUMN` | How OIDC `sub` claim maps to `users` rows: `email` (default) or `external_idp_subject` if you've migrated to that column. |
| `OAUTH_PROXY_ISSUER` | Optional override for the proxy's issuer URL in `.well-known` (default `APP_PUBLIC_URL`). |
| `OAUTH_PROXY_CALLBACK` | Optional override for `/oauth/callback` (default `${OAUTH_PROXY_ISSUER}/oauth/callback`). |

## OpenAPI v3 generic connector (v0.6.0)

A new connector type `openapi_v3` lets operators add REST-API source feeds without writing per-vendor sync code. There are no new env vars; everything is per-feed config in `source_feeds.connector_config_json`. See [ADR-0016](adr/0016-openapi-v3-generic-connector.md).

Constraints (v0.6.0): bearer auth only, offset/limit pagination only, JSONPath strict-mode (no filter expressions), 5MB spec cap, `record_type` closed to the 14 types in `chunks/extract.go`. Cursor + link-header pagination → v0.7.

## MCP endpoint (API, v0.5.1)

Mounts `/mcp` (Model Context Protocol streamable HTTP transport) with a small initial tool set: `kl_search`, `kl_ask_global`, `kl_get_entity`. Requires `OAUTH_PROXY_ENABLED=true` — bearer auth flows through the proxy and every tool call routes through `AccessEvaluator`. See [ADR-0015](adr/0015-oauth-proxy-and-mcp-bridge.md).

| Variable | Description |
|----------|-------------|
| `MCP_ENABLED` | `true` to mount `/mcp` (default `false`). Startup fails when set without `OAUTH_PROXY_ENABLED=true`. |

Operator example (Claude Desktop config):

```json
{
  "mcpServers": {
    "knowledge-layer": {
      "url": "https://kl.example.com/mcp"
    }
  }
}
```

## L1 cache (API, v0.4.0)

In-process BigCache for hot read paths. Off by default. Enable only on the API container — workers do not need it. The cache is **principal-scoped** (every key carries the requesting user) and invalidates on `entity.published`, `role.granted`, `policy.updated`, and `feed.config_patched` events. `/effective-access` cache-hit responses are at most `CACHE_L1_TTL_SECONDS` stale; **backend authorization decisions always run through `AccessEvaluator` regardless** — the cache controls UI affordances, not authz.

| Variable | Description |
|----------|-------------|
| `CACHE_L1_ENABLED` | `true` to enable in-process L1 cache (default `false`). |
| `CACHE_L1_TTL_SECONDS` | Default TTL for cached responses (default `60`). Lower this if operators expect faster propagation of permission changes to UI. |
| `CACHE_L1_MAX_MB` | BigCache memory ceiling (default `64`). Pre-allocated; budget alongside the API container memory limit. |

## Web

| Variable | Description |
|----------|-------------|
| `NEXT_PUBLIC_API_URL` | API origin for browser requests |
| `NEXT_PUBLIC_USE_DEV_HEADER` | When `true`, web sends `X-Principal-User-ID` (local pilot only). Must be `false` in staging/production with session auth. |
| `NEXT_PUBLIC_PRINCIPAL_USER_ID` | Dev-only principal UUID when using the dev header |
| `NEXT_PUBLIC_DOCS_BASE_URL` | Optional. GitHub UI base path ending at `/docs` (no trailing slash), e.g. `https://github.com/org/repo/blob/main/docs`. Enables in-app “read full doc” links in help callouts. |

In `session` mode, the web app should send `credentials: 'include'` for cookie auth.

## Retrieval and AI (API)

These control Ask, embeddings, and hybrid retrieval. For **what still works without them**, see [SELF_HOSTED.md](./SELF_HOSTED.md) (capability matrix).

| Variable | Description |
|----------|-------------|
| `OPENSEARCH_URL` | Empty: search uses Postgres title match for `q`; set: full-text discovery then permission filter. |
| `OPENROUTER_API_KEY` | If set, chat and embeddings go through OpenRouter (`OPENROUTER_BASE_URL`, default `https://openrouter.ai/api/v1`); `OPENAI_API_KEY` is not required. |
| `OPENROUTER_BASE_URL` | Optional; must include `/api/v1` when using OpenRouter’s default layout. |
| `OPENROUTER_TRANSCRIPTION_MODEL` | Model id for Ask voice notes on OpenRouter (chat + `input_audio`; default `openai/whisper-large-v3`). |
| `OPENAI_API_KEY` | When `OPENROUTER_API_KEY` is unset: required for real embeddings and chat (or use `OPENAI_MOCK=1`). |
| `OPENAI_TRANSCRIPTION_MODEL` | Whisper model for Ask `audio_base64` when using OpenAI (default `whisper-1`). |
| `OPENAI_BASE_URL` | Optional; default OpenAI API. Point to an on-prem compatible gateway for self-hosted models. |
| `OPENAI_MODEL` | Chat model for Ask / retrieval intelligence. |
| `OPENAI_EMBEDDING_MODEL` | Embedding model name when using default embed path. |
| `OPENAI_MOCK` | Set to `1` for tests or deterministic local behavior without network (see `internal/llm`). |
| `OPENAI_MOCK_OUTPUT` | Optional fixture output when mocking. |
| `RETRIEVAL_HYBRID_W_KEYWORD`, `RETRIEVAL_HYBRID_W_SEMANTIC`, `RETRIEVAL_HYBRID_PENALTY_WEIGHT` | Optional weights for hybrid retrieval (`internal/retrieval`). |

### Multimodal Ask (images, voice) and documents

- **Documents:** unchanged — **ingestion → text → chunks → search**; Ask still grounds on that text evidence.
- **Images:** `POST /ask` and `POST /entities/:id/ask` accept `images[]` with `url` (https or `data:image/...`) or `data_base64` + `media_type` (`image/*`). Up to **8** images are appended to the user message as `image_url` parts after the sanitized evidence text ([`internal/llm`](../apps/api/internal/llm), [`internal/qa`](../apps/api/internal/qa)).
- **Voice:** same endpoints accept `audio_base64` + `audio_format` (e.g. `wav`, `mp3`). The API **transcribes** first, merges text into `question`, then runs search/synthesis.  
  - **OpenAI:** `POST …/v1/audio/transcriptions` — model from **`OPENAI_TRANSCRIPTION_MODEL`** (default `whisper-1`).  
  - **OpenRouter:** transcription uses **`POST …/chat/completions`** with `input_audio` — model from **`OPENROUTER_TRANSCRIPTION_MODEL`** (default `openai/whisper-large-v3`). Pick an [audio-input model](https://openrouter.ai/models?fmt=cards&input_modalities=audio) if you override.
- **Embeddings** remain **text-only** (semantic/hybrid unchanged).

**TTS / audio output** from the model is not implemented (OpenRouter audio output is streaming-only upstream).

## AI privacy (API)

| Variable | Description |
|----------|-------------|
| `AI_PRIVACY_VAULT_KEY` | Production: 32-byte key for placeholder vault ([AI_REHYDRATION_LAYER.md](./AI_REHYDRATION_LAYER.md)). |
| `AI_PRIVACY_DEV_PLAINTEXT_STORE` | Local/tests only; never in production. |

## Second Brain overlay (optional)

| Variable | Description |
|----------|-------------|
| `SECOND_BRAIN_WEBHOOK_SECRET` | When set, exposes `POST /webhooks/second-brain/<secret>/{telegram,mattermost}` (no session). Use a long random value; rotate like any shared secret. |
| `TELEGRAM_BOT_TOKEN` | Telegram Bot API token for **outbound** replies (webhooks + pre-brief delivery). |
| `SECOND_BRAIN_BOT_SCENARIO_CODE` | Optional; when set, bot `/ask` uses the same `PrincipalAllowsScenario` gate as `POST /ask`. |
| `MATTERMOST_OUTGOING_WEBHOOK_TOKEN` | When set, Mattermost webhook must send matching `token` form field. |
| `SECOND_BRAIN_PREBRIEF_TICK` | Jobworker: set to `1` to run `ProcessPreBriefTick` after each `knowledge:scheduled_tick` (requires rows in `pre_meeting_brief_queue` and `TELEGRAM_BOT_TOKEN` and/or Redis). When Redis outbound is used, `sent_at` is set when the task is **enqueued** (delivery is best-effort in the worker). |
