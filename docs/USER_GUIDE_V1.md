# User guide (v1)

For **employees** using a self-hosted Knowledge Layer instance.

## What this is

A governed **organizational memory** workspace: search and open knowledge entities, see trust and lifecycle, and (where allowed) use Ask with citations. It is not a public internet chat or a full collaboration suite.

## Local golden path (evaluators)

Use this sequence on a fresh local instance to verify the product is coherent before exploring operators:

1. **Bootstrap** — If the API reports no workspace, open `/bootstrap` (or rely on API auto-bootstrap with `BOOTSTRAP_*` env vars).
2. **Home** — `/` loads your feed (may be sparse until data exists).
3. **Search** — `/search` runs permission-scoped queries (requires OpenSearch from compose to be up).
4. **Entity** — Open a hit → `/entities/{id}` for detail and trust signals.
5. **Ask** — `/ask` for cited answers over permitted hits (LLM/embeddings must be configured per [SELF_HOSTED.md](./SELF_HOSTED.md)).

What is **partial** today (connectors, job processors, setup wizards) is summarized in [LIMITATIONS.md](./LIMITATIONS.md). The in-app Home page repeats the short list for honesty.

## Getting access

1. You receive an **email invitation** from your administrator (no public registration).
2. Open the link, set your **name** and **password** (minimum 8 characters).
3. Sign in at `/login` when your instance uses session authentication.

If something fails, contact your admin — they manage domains, grants, and SMTP.

## Daily use

| Goal | Where to go |
|------|-------------|
| Search | **Search** (sidebar **Start**) — text query respects your domain grants |
| Browse by type | **Browse index** / type routes — decision, policy, SOP, etc. |
| Open an item | Click a card → entity detail |
| Home feed | **Home** — decisions, approved content, digests (when populated) |
| Governance tasks | **Governance** — if you review or approve content |

## Empty or sparse content

If lists are empty, your organization may still be **connecting sources** (Telegram, Drive, etc.) or **curating** content. Ask your admin about source feeds and jobs.

## Further reading

- [ADMIN_GUIDE_V1.md](./ADMIN_GUIDE_V1.md) — for people operating the instance
- [SELF_HOSTED.md](./SELF_HOSTED.md) — deployment (operators)
