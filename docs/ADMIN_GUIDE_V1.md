# Administrator guide (v1)

For operators who **deploy and run** a self-hosted instance.

## 1. Deploy

Follow [SELF_HOSTED.md](./SELF_HOSTED.md): Postgres, API, web, optional OpenSearch, env vars.

## 2. First workspace

- **Automatic:** On API start, if there are **no domains**, the server can create the first admin + workspace (see `AUTO_BOOTSTRAP_*` in [CONFIG_ENV.md](./CONFIG_ENV.md) and [SELF_HOSTED.md](./SELF_HOSTED.md)). Local defaults apply unless `AUTO_BOOTSTRAP_INSTANCE=0`.
- **Manual:** Open **`/bootstrap`** in the web app and submit the form (admin user + workspace name), or call `POST /instance/bootstrap`.
- Or use development seed + `AUTH_MODE=development_header` for local pilots only.

## 3. Authentication

| Mode | When to use |
|------|-------------|
| `development_header` | Local dev; web sends `X-Principal-User-ID` when `NEXT_PUBLIC_USE_DEV_HEADER=true` |
| `session` | Production: set `SESSION_SECRET`, `AUTH_MODE=session`, `NEXT_PUBLIC_USE_DEV_HEADER=false`, `CORS_ALLOW_ORIGINS` to your web origin |

Set `APP_PUBLIC_URL` to the **web** origin used in invitation links (e.g. `https://knowledge.company.com`).

## 4. Email (invitations)

Configure `SMTP_*` variables. Use **Instance settings → Test email** in the UI, or watch API logs if SMTP is unset (messages are logged).

## 5. Users

- **Single invitation:** `POST /invitations` (JSON: `email`, `domain_id`, optional `role_id`, `access_level`, `sensitivity_cap`).
- **Bulk:** `POST /users/import` with multipart field `file` (CSV). Required columns: `email`, `domain_id`. Optional: `name`, `access_level`, `sensitivity_cap`, `role_id`. Form fields: `mode=invite` (default) or `mode=active`, `send_invites=true` to email each row.

## 6. Domains

- **Create:** `POST /domains` or additional workspaces after bootstrap.
- **Update:** `PATCH /domains/:id` (requires publish on that domain).

## 7. Connectors and jobs

- **Connectors** and **Source feeds** in the UI: connect systems, then activate/sync.
- **Jobs** — scheduled or manual knowledge operations; see [KNOWLEDGE_JOBS.md](./KNOWLEDGE_JOBS.md).

## 8. Operations

See [OPERATIONS.md](./OPERATIONS.md) for health checks and failed runs.

## Configuration reference

- [CONFIG_ENV.md](./CONFIG_ENV.md)
