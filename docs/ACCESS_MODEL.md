# Access model (v1)

## Authentication (production)

- **`AUTH_MODE=development_header`**: principal is taken only from `X-Principal-User-ID`. **Allowed only when `APP_ENV` is not `staging` or `production`** (local / dev profiles). The API refuses to start with `development_header` in staging or production.
- **`AUTH_MODE=session`**: principal is taken from signed session cookie `kl_session` after login or invitation accept; `SESSION_SECRET` is required (min effective length per [`SessionSecretBytes`](../apps/api/internal/config/config.go)). **Required** for `APP_ENV=staging` or `production` (enforced at startup).
- **Session cookies**: `Secure` defaults to **true** when `APP_ENV` is staging or production (override with `SESSION_COOKIE_SECURE=false` only for HTTP-only **non-production** lab setups; **production** startup rejects insecure cookies).
- **`ALLOW_SELF_REGISTRATION`**: defaults to `false`. Users are created by admins or by accepting an **invitation** (email). There is no public sign-up unless this flag is explicitly enabled.

See [SELF_HOSTED.md](./SELF_HOSTED.md). Permission resolution detail: [permission-system.md](./permission-system.md) (canonical) and [permission-flow.md](./permission-flow.md). Production posture: [PRODUCTION_HARDENING.md](./PRODUCTION_HARDENING.md).

## Operations and metrics HTTP

- **`GET /health`**: Public liveness; no dependency checks.
- **`GET /ops/health`**: **Local**: detailed JSON (may include raw dependency errors). **Staging** with `OPS_AUTH_TOKEN` **unset**: **200** with **redacted** booleans only (`database_ok`, `opensearch_ok`). **Staging** with token set, or **production** (token required at startup): **401** without `Authorization: Bearer <OPS_AUTH_TOKEN>`; valid bearer returns detailed JSON. See [`routes_health.go`](../apps/api/internal/httpserver/routes_health.go).
- **`GET /metrics`**: Stub text in **local** dev without auth. **Staging/production**: **401** unless `OPS_AUTH_TOKEN` is **non-empty** and the request sends a matching bearer token (if the token is unset in staging, `/metrics` is always **401**).
- **`GET /ops/failed-runs`**: Still requires an authenticated principal with identity-admin capability (unchanged).

## Surfacing vs access grants

- **`user_scope_follows`** (see [DOMAIN_MODEL.md](./DOMAIN_MODEL.md)) records what a user wants to **see first** on Home (followed domains, hubs, knowledge topics, digest streams). It is **not** a substitute for `domain_grants`, roles, or entity-level policy: the API still evaluates `view` / `publish` before returning entities or running retrieval.

## Principles

- **Deny by default**: no grant means no domain scope for retrieval and most mutations.
- **Policy evaluation** is centralized in `identity_access.AccessEvaluator.Evaluate` and wrapped by `platform/permissions.Resolver`; HTTP handlers and retrieval call it before sensitive operations.
- **Entity type scope**: after domain grants, policies may restrict which `entities.type` values apply (`access_policies.entity_type_scope`). An explicit `entity_acl` **allow** can bypass type restriction for `view` / `search` only.
- **Object-level ACL**: `entity_acl` rows with `effect = deny` remove access for the given user or team principal even when domain grants would otherwise allow it.
- **Granted domains for retrieval**: `search` uses `DomainIDsWithGrant` for SQL scoping, then **per-hit** `Evaluate(view)` so titles/snippets cannot leak past entity ACL, type policy, or sensitivity.
- **Scenario binding (retrieval):** optional non-empty `scenario_code` on **`GET /search`** and **`POST /ask`** is checked via Role Builder scenario bindings (`PrincipalAllowsScenario`); empty omits the gate. **`POST /entities/:id/ask`** has no scenario field — only entity `view` and scoped retrieval apply. See [AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md) §4.
- **Audit**: identity admin mutations write audit events (`user.created`, `domain_grant.*`, `role_binding.*`, `team.*`, etc.). Preset catalog instantiation emits `preset.instantiated`; successful onboarding launch emits `onboarding.launched`.
- **Role Builder**: reusable role definitions (`roles` + binding tables) and assignments (`user_role_bindings`). See [role-builder.md](./role-builder.md). Evaluator respects active roles, optional **role domain allowlists**, and **role entity-type unions** in addition to domain policies.

## Identity admin HTTP API

In `development_header` mode these routes use `X-Principal-User-ID`. In `session` mode the same gates apply using the authenticated user id from the session cookie.

### Gate: `requireCanManageIdentity`

List/read and most mutations require the principal to have **`publish`** on **at least one** domain they are granted (seed admin pattern).

### Gate: `requirePublishOnDomain`

Mutations that affect a specific domain require **`publish`** on **that** domain:

- `POST /domain-grants`, `PATCH /domain-grants/:id`, `DELETE /domain-grants/:id`
- `POST /user-role-bindings` when `scope_type` is `domain` (and for `PATCH`/`DELETE` when binding is domain-scoped)

### Endpoints

| Method | Path | Notes |
|--------|------|--------|
| POST | `/users` | Create user (identity admin gate) |
| PATCH | `/users/:id` | Update user |
| GET | `/roles` | List roles (enriched: `code`, `category`, `active`, `scope_model`, `is_preset`, …) — Role Builder |
| GET | `/teams` | List teams (`limit`, `offset`) |
| POST | `/teams` | Create team |
| PATCH | `/teams/:id` | Update team |
| GET | `/user-team-memberships` | Filter `team_id`, `user_id` |
| POST | `/user-team-memberships` | Add membership |
| DELETE | `/user-team-memberships/:id` | Remove membership |
| GET | `/domain-grants` | Filter `user_id`, `domain_id` |
| POST | `/domain-grants` | Upsert grant (`user_id`, `domain_id`, `access_level`, `sensitivity_cap`) |
| PATCH | `/domain-grants/:id` | Update grant |
| DELETE | `/domain-grants/:id` | Remove grant |
| GET | `/user-role-bindings` | Filter `user_id` |
| POST | `/user-role-bindings` | Create binding (`scope_type` default `global`) |
| PATCH | `/user-role-bindings/:id` | Update binding |
| DELETE | `/user-role-bindings/:id` | Remove binding |

### Role Builder (same gates)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/roles/presets` | Preset catalog |
| POST | `/roles` | Create definition + bindings |
| POST | `/roles/from-preset` | Materialize from `preset_key` |
| GET | `/roles/:id` | Full definition |
| PATCH | `/roles/:id` | Metadata; optional `bindings` full replace |
| DELETE | `/roles/:id` | Soft-deactivate if assignments exist |
| POST | `/roles/:id/clone` | Duplicate definition |
| GET | `/roles/:id/assignments` | List bindings for role |
| POST | `/roles/:id/assignments` | Assign role to user |
| GET | `/roles/:id/preview` | Structured effective-access preview |

Details: [role-builder.md](./role-builder.md).

### Core API listing and evaluation (authenticated)

- **`GET /users`** — requires authenticated principal and **`requireCanManageIdentity`** (same gate as identity admin: `publish` on at least one granted domain). Handler: [`routes_register.go`](../apps/api/internal/httpserver/routes_register.go).
- **`GET /users/:id`** — same gate; returns one user row.
- **`GET /domains`** — requires authenticated principal; returns **only domains with a non-expired `domain_grants` row** for that user (`identity_access.Repo.ListDomainsForUser`). Not a global domain listing for non-admins.
- **`POST /access/evaluate`** — requires authenticated principal; **`principal_id` in the JSON body is required** and must equal the caller **unless** the caller passes **`requireCanManageIdentity`** (then admins may evaluate other principals). Prevents unauthenticated permission probing.
- **`POST /domains/:id/apply-setup-kit`** — records an audit event and returns `{ "recorded": true, "applied": false, "kit": { ... } }`. Kit payload is documentation-oriented; it does **not** create roles/jobs in the database (see `onboarding.DomainSetupKit`). Requires `publish` on the target domain.

## Bootstrap and invitations (HTTP)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/instance/status` | Public: `needs_bootstrap`, `domain_count`, `auth_mode` |
| POST | `/instance/bootstrap` | One-time when no domains; creates admin + domain + grants (same outcome as optional startup auto-bootstrap — see [CONFIG_ENV.md](./CONFIG_ENV.md) `AUTO_BOOTSTRAP_*`) |
| POST | `/auth/login` | Session mode: sets `kl_session` cookie |
| POST | `/auth/logout` | Clears session cookie |
| GET | `/auth/me` | Current user when authenticated |
| POST | `/auth/register` | Disabled unless `ALLOW_SELF_REGISTRATION=true` (returns 403 by default) |
| POST | `/invitations` | Admin: create invitation + email link |
| GET | `/invitations/preview` | Public: validate token, show email |
| POST | `/invitations/accept` | Public: set password, grants, session |
| POST | `/users/import` | Admin: multipart CSV bulk invite or active users |

## Knowledge jobs (HTTP)

Job list/detail/patch and run metadata use the job’s **`output_domain_id`** with `identity_access.Evaluate` on resource type **`domain`**: **`view`** or **`manage_jobs`** for reads/updates; **`POST /knowledge-jobs/:id/run`** allows the job **owner**, rows in **`knowledge_job_operators`**, or domain **`run_job`** / **`manage_jobs`**. If `output_domain_id` is null, non-owners are denied for domain-scoped routes (fail closed). See [knowledge-jobs-engine.md](./knowledge-jobs-engine.md) and [permission-system.md](./permission-system.md).

## UI

**Users and Access** surface: `apps/web/src/app/(dash)/access/page.tsx` (dev header or session). **Role Builder** scaffold: `apps/web/src/app/(dash)/admin/roles/page.tsx`. **Bulk import** CSV on the access page. **Login / invite / bootstrap / settings** under dedicated routes in `apps/web/src/app/`.
