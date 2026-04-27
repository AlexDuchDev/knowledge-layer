# Secret rotation runbook

How to rotate every secret Knowledge Layer holds without dropping traffic. Read [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md) first for the full secret inventory.

## TL;DR

- **`SESSION_SECRET`**: short downtime acceptable (existing sessions invalidate). Rotate once a quarter or after a leak.
- **`OPS_AUTH_TOKEN`**: zero-downtime via dual-token window — see §3.
- **`AI_PRIVACY_VAULT_KEY`**: NOT a simple swap — see §4 (vault data is encrypted with the key; rotation requires re-encryption or a key-rolling window).
- **OAuth / LLM API keys**: provider-side rotation, then update the env. See §5.
- **`DATABASE_URL`** / **`REDIS_URL`** passwords: provider-specific; rotate the credential, then redeploy with the new URL.

## 1. Secret inventory

| Secret | Where it lives | Operational impact of rotation |
|---|---|---|
| `SESSION_SECRET` | API + worker env | All active sessions invalidate; users must re-login. ~5–60s API restart per pod. |
| `OPS_AUTH_TOKEN` | API + worker env | Without dual-token window: scrapers and `/ops/health` clients 401 until they refresh. |
| `AI_PRIVACY_VAULT_KEY` | API + worker env | Existing vault rows can no longer be decrypted with the new key — **must re-encrypt or accept loss of in-flight Ask traces**. |
| `OPENAI_API_KEY` / `OPENROUTER_API_KEY` | API + worker env | Old key still works during the provider's grace period; rotate provider-side then env. |
| Connector OAuth (`GMAIL_OAUTH_*`, `MICROSOFT_OAUTH_*`) | API env | Existing refresh tokens may need re-consent if the OAuth app's client secret rotates; provider-dependent. |
| Slack `signing_secret` per feed | `connector_config_json` (Postgres) | New deliveries reject as "bad signature" until you update the feed config; Slack re-tries for 30 min. |
| `DATABASE_URL` / `REDIS_URL` passwords | API + worker env | Connection pool drops; restart picks up the new URL. |
| `SECOND_BRAIN_WEBHOOK_SECRET` | API env | Inbound Telegram/Mattermost webhooks reject until updated; outbound delivery unaffected. |
| `TELEGRAM_BOT_TOKEN`, `MATTERMOST_OUTGOING_WEBHOOK_TOKEN` | API + jobworker env (Second Brain only) | Outbound delivery fails until env updated. |
| `BLOBSTORE_S3_*` | API + connectorworker env | New writes/reads fail; rotate provider-side then env. |
| `BOOTSTRAP_ADMIN_PASSWORD` | API env (one-shot) | Only consumed at first-run auto-bootstrap; remove after the admin sets a real password via the UI. |

## 2. `SESSION_SECRET`

Cookie sessions sign with one secret at a time. Rotation is destructive — pick a maintenance window or accept the re-login storm.

```bash
NEW=$(openssl rand -hex 32)

# Update the env (Kubernetes example)
kubectl -n knowledge-layer create secret generic knowledge-layer-secrets \
  --from-literal=SESSION_SECRET="$NEW" \
  --dry-run=client -o yaml \
  | kubectl apply -f -          # only the SESSION_SECRET key changes

# Roll the API and workers
kubectl -n knowledge-layer rollout restart deployment/knowledge-api deployment/knowledge-jobworker deployment/knowledge-connectorworker
```

After rotation: existing sessions are unsignable → users re-login on next request. Verify by tailing the API and watching the spike in `POST /auth/login` succeed.

A future hardening (Phase 4) may introduce a dual-secret window so the API trusts both old and new for a short overlap; until then, rotation = brief logout for everyone.

## 3. `OPS_AUTH_TOKEN` (dual-token window)

Scrapers (Prometheus, uptime checks) authenticate against `/ops/health` and `/metrics` with a bearer. To rotate without a scrape gap:

### Step 1 — Generate the new token

```bash
NEW=$(openssl rand -hex 24)
echo "$NEW"   # capture it — you'll need it to update scrapers
```

### Step 2 — Add the new token alongside the old

The current implementation reads a single `OPS_AUTH_TOKEN`. Until a multi-token comparator ships (Phase 4 follow-up), use a **provider-side bridge**:

- If your secret store supports versioned references, point Prometheus and uptime checks at the next version *before* rotating the env.
- For ingress-controller-level auth (some stacks proxy bearer-validation outside the API), add the new token at the proxy first, then swap the API env.
- Otherwise accept a brief 401 window: update the API env and the scraper at the same time.

### Step 3 — Update API and worker env

```bash
kubectl -n knowledge-layer create secret generic knowledge-layer-secrets \
  --from-literal=OPS_AUTH_TOKEN="$NEW" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n knowledge-layer rollout restart deployment/knowledge-api deployment/knowledge-jobworker deployment/knowledge-connectorworker
```

### Step 4 — Verify

```bash
curl -s -H "Authorization: Bearer $NEW" https://kl.example.com/ops/health
curl -s -H "Authorization: Bearer $NEW" https://kl.example.com/metrics | head
# Both should return 200 with substantive content.
```

Update Prometheus / uptime probes to use `$NEW`. Discard the old token.

## 4. `AI_PRIVACY_VAULT_KEY` (re-encryption required)

The vault stores per-correlation placeholder mappings encrypted with this AES-256 key. After rotation, **rows encrypted with the old key are unreadable**. There are three paths:

### Path A — Drain + rotate (simplest, brief window)

1. Stop the workers (no new vault writes from job processors).
2. Wait for in-flight Asks to complete (or 60 s — answer-trace rehydration is async).
3. Rotate the env on API + workers; restart.
4. Restart workers.
5. Existing vault rows for completed Asks are now unreadable; no live request needs them, so the loss is bounded to "answer-trace rehydration of historical asks".

```bash
# Stop workers
kubectl -n knowledge-layer scale deploy/knowledge-jobworker --replicas=0
kubectl -n knowledge-layer scale deploy/knowledge-connectorworker --replicas=0

# Generate + apply new key
NEW=$(openssl rand -base64 32)
kubectl -n knowledge-layer create secret generic knowledge-layer-secrets \
  --from-literal=AI_PRIVACY_VAULT_KEY="$NEW" \
  --dry-run=client -o yaml | kubectl apply -f -

# Roll
kubectl -n knowledge-layer rollout restart deployment/knowledge-api
kubectl -n knowledge-layer scale deploy/knowledge-jobworker --replicas=1
kubectl -n knowledge-layer scale deploy/knowledge-connectorworker --replicas=1

# Optional cleanup of unreadable historical rows (TTL-based pruning happens
# anyway via the vault's expiration logic, but you can force it):
psql "$DATABASE_URL" -c "DELETE FROM ai_placeholder_mappings WHERE expires_at < now() - interval '1 day';"
```

Acceptable when: the loss of cleartext for historical Ask traces is OK (the original Ask outputs are immutable in `answer_traces`; only the *rehydration* step loses fidelity).

### Path B — Re-encrypt in place (no loss, more work)

Requires a one-shot re-encryption job. Knowledge Layer does not ship one in v1; if you need this, file an issue describing your retention requirement so the maintainers can add it. The shape would be:

1. Read all `ai_placeholder_mappings` rows.
2. Decrypt each with the old key.
3. Re-encrypt with the new key.
4. Update the row in a transaction.

Until that lands, prefer Path A.

### Path C — Forbidden: dropping the vault

Setting `AI_PRIVACY_DEV_PLAINTEXT_STORE=1` in production is rejected at startup (`ValidateAPI` / `ValidateWorker` — Phase 1.1.2). Don't try to "skip" rotation by switching to plaintext. The fail-closed is intentional.

## 5. LLM provider keys (`OPENAI_API_KEY`, `OPENROUTER_API_KEY`)

Provider-side rotation usually leaves the old key valid for a grace period. Rotation is then:

1. Generate a new key in the provider dashboard.
2. Verify the new key (provider's CLI or a one-off curl).
3. Update the API + workers env, restart.
4. Revoke the old key in the provider dashboard once you've seen successful Asks/embeddings on the new key.

```bash
NEW='sk-...'
kubectl -n knowledge-layer create secret generic knowledge-layer-secrets \
  --from-literal=OPENAI_API_KEY="$NEW" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n knowledge-layer rollout restart deployment/knowledge-api deployment/knowledge-jobworker deployment/knowledge-connectorworker
```

## 6. Connector secrets (per-feed)

Slack `signing_secret`, OAuth client secrets, and similar live in `source_feeds.connector_config_json` rather than env. Rotation is a **PATCH on the feed**, no restart required:

```bash
# Slack example: rotate the signing secret in the Slack app dashboard, then update the feed.
curl -s -X PATCH https://kl.example.com/source-feeds/<feed-id> \
  -H "Authorization: Bearer $SESSION_BEARER_OR_DEV_HEADER" \
  -H "Content-Type: application/json" \
  -d '{"connector_config":{"signing_secret":"<new-secret>"}}'
```

For OAuth flows, the user-side refresh token usually survives a client-secret rotation; if not, the feed shows `health_status="degraded"` and the operator re-authorizes via the source-feed wizard.

## 7. Database / Redis credentials

Provider-specific (managed Postgres / Redis support credential rotation natively).

```bash
# Rotate provider-side, then:
kubectl -n knowledge-layer create secret generic knowledge-layer-secrets \
  --from-literal=DATABASE_URL='postgres://knowledge:NEW_PASSWORD@db.internal:5432/knowledge?sslmode=require' \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n knowledge-layer rollout restart deployment/knowledge-api deployment/knowledge-jobworker deployment/knowledge-connectorworker
```

The pgx connection pool reconnects on the next health probe. Check API + worker `/ops/health` afterwards.

## 8. Verification after every rotation

Mandatory:

- [ ] `curl -s https://kl.example.com/health` returns 200.
- [ ] `curl -s -H "Authorization: Bearer $OPS_AUTH_TOKEN" https://kl.example.com/ops/health` returns 200 with `database:"ok"`.
- [ ] `kubectl -n knowledge-layer logs deploy/knowledge-api --since=5m` shows no `ValidateAPI` startup errors.
- [ ] Workers' `/ops/health` (`:9001`, `:9002`) shows `last_processed_by_task` advancing within a minute.
- [ ] One end-to-end Ask returns an answer with citations (validates LLM key + vault).
- [ ] `audit_events` shows expected types from the verification Ask (`vault.placeholder_stored`, `vault.placeholder_decrypted`, `vault.rehydration_applied` — Phase 1.1.3 hardening).

## 9. Rotation cadence (recommended)

| Secret | Cadence |
|---|---|
| `SESSION_SECRET` | Quarterly, plus immediately after any suspected leak. |
| `OPS_AUTH_TOKEN` | Quarterly. |
| `AI_PRIVACY_VAULT_KEY` | Annually + immediately after compromise. Plan the maintenance window per Path A above. |
| LLM API keys | Quarterly + immediately if the key surface (logs, error reports) might have leaked. |
| Connector OAuth client secrets | Per provider's policy (Google/Microsoft typically yearly). |
| Per-feed Slack `signing_secret` | When the Slack app owner rotates; no schedule otherwise. |
| Database / Redis | Per managed-provider policy. |

Document executed rotations in your team's runbook log. Knowledge Layer does not auto-record secret rotations — that's intentional, since the secret values themselves should never be in audit storage.

## Related

- [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md) — startup validation rules each rotation must satisfy.
- [CONFIG_ENV.md](./CONFIG_ENV.md) — full env-var reference.
- [UPGRADE_AND_ROLLBACK.md](./UPGRADE_AND_ROLLBACK.md) — rotation often piggy-backs on a release; coordinate.
- [OPERATIONS.md](./OPERATIONS.md) — health endpoints + audit observability used by the verification step.
