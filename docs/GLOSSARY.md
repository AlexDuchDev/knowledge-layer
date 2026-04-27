# Glossary

Canonical terms for the Knowledge Layer platform. When a term changes in code or UI, update this file and [DOMAIN_MODEL.md](DOMAIN_MODEL.md) / [DOMAIN_MODEL_CONTRACT.md](DOMAIN_MODEL_CONTRACT.md) as appropriate (see [DOCS_MAINTENANCE_POLICY.md](DOCS_MAINTENANCE_POLICY.md)).

| Term | Definition |
|------|------------|
| **Connector** | A **plugin type** registered in the platform (e.g. Slack, Telegram) that knows how to talk to an external system. Connectors are **not** the configured instance—see **Source Feed**. |
| **Source Feed** | A **governed configuration** that binds a connector to a domain, owner, sensitivity, sync mode, and scope. Feeds produce **raw artifacts** and drive ingestion. |
| **Raw Artifact** | Immutable (or versioned) payload and metadata captured from a source before full normalization—preserved for audit and replay. |
| **Normalized Artifact** | Structured record derived from raw input, ready for linking and entity materialization per connector rules. **Preferred implementation term:** **normalized record** (`normalized_records` table, `GET /normalized-records/:id`). |
| **Canonical Entity** | A durable **knowledge object** in the core model (title, type, domain, lifecycle, truth mode, policies) that users search and govern—not “just a file.” |
| **Role** | A **reusable access template** (permissions pattern) that can be assigned to users; combines with domain grants and policies. |
| **Scenario** | A **timed or triggered context** that binds roles, sources, and jobs—defines *when* and *what* runs together (e.g. weekly digest scenario). |
| **Job** | A **first-class configured task** (knowledge job) with triggers, scope, and output policy—executed by the jobs engine/worker. |
| **Preset** | A **catalog entry** that seeds roles, scenarios, or jobs for faster setup; instantiation creates editable live objects. |
| **Setup Session** | A **guided first-run** flow in the control plane: templates, preview, launch—distinct from day-2 admin tweaks. |
| **Digest** | A **recurring summary** job output (e.g. weekly) scoped to sources and policies—often tied to a digest-type job. |
| **Decision** | A **canonical entity** (or view) representing an organizational decision record with lifecycle and governance. |
| **Governance** | The **set of controls** over quality, risk, and publication: reviews, approvals, policy exceptions, owners, stale content. |
| **Review** | A **human checkpoint** on content or outputs before wider visibility (queue-driven). |
| **Approval** | An explicit **authorize step** (often stricter than review) for high-authority or sensitive publication. |
| **Ask** | **Question answering** over permitted corpus with **citations** and trace—not unconstrained chat. |
| **Search** | **Keyword / scoped retrieval** over entities the principal may view—filtered by domain grants and sensitivity. |
| **Explorer** | **Browse and drill-down** surfaces over memory (entities, relationships) within permission scope. |
| **Project Memory** | **Curated or scoped view** of knowledge tied to a project (or workspace metaphor)—still governed by grants. |
| **Output Policy** | Rules for **what the system may produce** (e.g. draft vs publish, redaction) for AI or job outputs. |
| **Sensitivity Level** | Numeric (or enumerated) **classification** on feeds/entities influencing who may see content and which jobs may run. |
| **Domain** | Top-level **organizational partition** for grants, sources, and entities—primary scope boundary for retrieval. |

**See also:** [PRODUCT_CONCEPTS.md](PRODUCT_CONCEPTS.md), [CONTROL_PLANE_OVERVIEW.md](CONTROL_PLANE_OVERVIEW.md), [USER_GUIDE.md](USER_GUIDE.md).
