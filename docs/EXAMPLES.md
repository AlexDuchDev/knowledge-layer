# Examples

Copy-paste and reference examples for integrators and operators. JSON shapes are illustrative—verify against [API_SURFACE_V1.md](API_SURFACE_V1.md) and live API for your version.

## Ask (global)

```http
POST /ask
Content-Type: application/json

{
  "question": "What did we decide about Q3 budget?",
  "include_related": true,
  "answer_strategy": "standard",
  "domain_id": "00000000-0000-0000-0000-000000000000"
}
```

Response includes `answer`, `citations[]`, `supporting_entities[]`, `trace_id` for feedback and audit.

## Weekly digest scenario (conceptual)

A **scenario** of type suitable for digest binds:

- **Roles** that define who receives or approves output.
- **Source feeds** that supply content for the digest window.
- **Jobs** (e.g. `weekly_digest`) with `source_scope` pointing at feeds.

See [KNOWLEDGE_JOBS.md](KNOWLEDGE_JOBS.md) and [scenario-builder.md](scenario-builder.md).

## Planning summary flow

1. Ingest meeting notes or docs via **source feed**.
2. Normalization produces entities; **governance** queues catch high-sensitivity items.
3. **Ask** over the project domain for “summary of decisions this week” with citations.
4. Optional **job** publishes a digest to stakeholders per **output_policy**.

## Governance review vs approval

- **Review** — human reads/changes requested before wider visibility.
- **Approval** — explicit authorize for publication or policy exception.

UI: `/governance/*` and `/control-plane/governance/*`. See [GLOSSARY.md](GLOSSARY.md).

## Preset instantiation

Use control plane **Presets** → pick role/scenario/job preset → instantiate → edit live object. See [preset-catalog.md](preset-catalog.md).

## Role / scenario / job JSON (illustrative)

Prefer creating via UI or documented POST bodies in [API_SURFACE_V1.md](API_SURFACE_V1.md). Example **job** fields you may see:

- `trigger_type`, `publication_mode`, `output_sensitivity_level`, `source_scope` (JSON).

## Connector vs source feed

- **Connector** row: type + capabilities (plugin).
- **Source feed** row: domain, owner, `knowledge_scope`, `sensitivity_level`, `sync_mode`, `connector_config_json`.

See [SOURCE_FEED_SETUP_FLOW.md](SOURCE_FEED_SETUP_FLOW.md).

## More setup examples

[SETUP_EXAMPLES.md](SETUP_EXAMPLES.md) — env snippets, DigitalOcean, and smoke commands.
