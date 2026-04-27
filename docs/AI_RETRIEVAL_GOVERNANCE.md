# AI_RETRIEVAL_GOVERNANCE.md

**Implementation reference:** the concrete schema, packages, worker task, and `POST /ask` retrieval modes for the retrieval foundation are documented in [retrieval-ai-foundation.md](./retrieval-ai-foundation.md).

## 1. Purpose

This document defines how retrieval, AI-assisted synthesis, and trust governance work in the Organizational Memory & Knowledge Operations Platform.

It establishes:
- supported retrieval modes
- retrieval architecture expectations
- permission-aware retrieval rules
- hybrid search behavior
- scoped Q&A behavior
- citation and evidence requirements
- trust indicators
- answer trace requirements
- freshness-aware retrieval rules
- AI task boundaries
- publication and review constraints for AI-generated outputs

This document is a core trust contract for the system.

---

## 2. Why this layer matters

The product is not just a governed storage system.
It becomes useful when users can:
- find the right knowledge quickly
- ask questions in natural language
- understand decisions and process context
- trust what they are seeing
- know when an answer is partial, stale, or derived

The danger is obvious:
if AI answers feel useful but are not governed, the product becomes risky.

This layer must therefore make AI useful **because it is constrained**, not despite constraints.

---

## 3. Core principles

### 3.1 AI is not an authority layer
AI may summarize, extract, synthesize, and suggest.
It does not define truth on its own.

### 3.2 Retrieval happens inside permissions
The system must only retrieve content that the requesting principal is allowed to access.

### 3.3 Context passed to AI must already be scoped
The model must never see hidden content and rely on post-filtering later.

### 3.4 Answers must be evidence-backed
Important answers should include citations or supporting entities.

### 3.5 Provenance matters as much as fluency
A polished answer without provenance is lower value than a slightly rougher answer with clear support.

### 3.6 Trust state must be visible
Users should understand whether an answer is:
- based on approved content
- based on derived artifacts
- stale
- partial due to access limits
- linked to canonical objects
- based on mirrored external sources

### 3.7 Generated outputs are governed artifacts
If AI creates a reusable artifact, it must enter the system with:
- owner
- domain
- sensitivity
- truth mode
- provenance
- lifecycle state
- policy source

### 3.8 Fail closed
If scope, access, provenance, or evidence handling is unclear, the system should narrow the result, hold for review, or deny.

---

## 4. Retrieval goals

The retrieval layer must help users:
- find exact knowledge quickly
- navigate related context
- reconstruct decisions and rationale
- search across different source types
- ask natural-language questions safely
- retrieve within allowed scope only
- see why an answer should be trusted

It must also help the platform:
- provide strong evidence to AI tasks
- support ranking and relevance
- preserve explainability
- respect workflow and freshness state
- remain operationally scalable

### 4.1 Scenario code on Ask, Search, and entity-scoped Ask

`POST /ask`, **`POST /entities/:id/ask`**, and `GET /search` accept an optional scenario identifier (`scenario_code` **JSON** field on both Ask routes; **`scenario_code` query param** on Search). When non-empty, the API **fails closed** (`403`) unless the principal has at least one non-expired `user_role_binding` to a role that lists that scenario in `role_scenario_bindings.scenario_key` (mirrored from scenario builder). Omitted or empty `scenario_code` keeps prior behavior (permission-scoped retrieval and entity **`view`**; no extra scenario gate).

**Entity-scoped Ask:** the same optional `scenario_code` JSON field applies **after** entity `view` checks. Evidence and synthesis behavior are unchanged; the scenario gate is an additional operator-controlled bundle when callers supply a code.

---

## 5. Supported retrieval modes

The system should support multiple retrieval modes because company knowledge is heterogeneous.

### 5.1 Exact / keyword retrieval

Use for:
- exact terms
- policy names
- project names
- known titles
- identifiers
- acronyms
- precise phrase matches

Good for:
- users who know what they are looking for
- authoritative object retrieval
- exact policy or SOP lookup

### 5.2 Filtered retrieval

Use for:
- narrowing by domain
- narrowing by entity type
- narrowing by sensitivity
- narrowing by owner
- narrowing by lifecycle state
- narrowing by freshness

Good for:
- admin and governance workflows
- structured browsing
- audit and review flows

### 5.3 Semantic retrieval

Use for:
- concept-level similarity
- paraphrased questions
- fuzzy decision context
- related summaries
- cross-source synthesis inputs

Good for:
- natural-language Q&A
- fuzzy context reconstruction
- insight and decision discovery

### 5.4 Hybrid retrieval

Use for:
- combining exact signals and semantic similarity
- ranking across both authoritative metadata and text meaning

This should be the default for many higher-value queries.

### 5.5 Relation-aware retrieval

Use for:
- expanding from a visible object to related objects
- understanding decision impact
- navigating meeting -> decision -> project -> process links

Important:
every relation expansion must re-check permissions.

### 5.6 Freshness-aware retrieval

Use for:
- preferring current approved content over stale content
- surfacing stale state rather than silently hiding it
- ranking active or confirmed artifacts appropriately

### 5.7 Permission-aware retrieval

This is not just another mode.
It is a mandatory constraint across all retrieval behavior.

### 5.8 Scope-first retrieval (facets before semantic breadth)

For search and Ask, the system should apply **allowed-scope constraints and structured facets** (domain, entity type, lifecycle, freshness, approval where applicable) **before or alongside** broad semantic retrieval, not only as a post-filter on a large candidate pool.

**Why:** Metadata-aligned narrowing reduces false positives and matches how enterprise users think (which domain, which artifact class, which trust state). It complements permission resolution (§7): permissions define *what may be queried*; facets define *which slice of the allowed corpus* this query targets.

**Ordering expectation:**
1. Resolve the principal and compute allowed retrieval scope (§7.1).
2. Apply facet filters and keyword/structured signals within that scope.
3. Run semantic or hybrid ranking on the constrained candidate set (or merge constrained sets), then apply trust-aware reranking (§8).

**Note:** Third-party benchmark scores on public conversational-memory datasets are **not** operational guarantees for permissioned organizational corpora. Use them only to inform **internal** offline evaluation (§23.3), not customer-facing SLAs.

---

## 6. Retrieval object model

The retrieval layer does not retrieve only files.
It retrieves a mix of governed objects and derived retrieval structures.

### 6.1 Primary retrieval targets

The system may retrieve:
- canonical entities
- mirrored entities
- derived artifacts
- entity versions
- chunks
- source-linked summaries
- reviewable outputs where permitted

### 6.2 Retrieval evidence structure

A retrieval result should ideally include:
- object reference
- entity type
- title
- snippet or evidence excerpt
- owner
- domain
- sensitivity if displayable
- lifecycle state
- truth mode
- freshness status
- citation anchor or supporting refs

### 6.3 Retrieval truth rule

A retrieval hit must not hide the object’s truth mode.
A derived artifact should not look like a canonical policy.

---

## 7. Permission-aware retrieval model

This is one of the hardest system requirements.

### 7.1 Required sequence

For any search or Q&A request, the system must:

1. identify the principal
2. validate principal status
3. resolve allowed domains
4. resolve allowed entity types
5. resolve action permissions
6. resolve sensitivity limits
7. apply object-level overrides
8. compute allowed retrieval scope
9. query only within that scope
10. rank only allowed results
11. assemble model context only from allowed results

### 7.2 What is forbidden

The system must not:
- retrieve broadly and filter later
- use hidden chunks as latent context
- cache broad result sets and reuse them across principals
- allow relation expansion to bypass access
- use citations as a substitute for access control

### 7.3 Partial-answer behavior

If permission scoping restricts the answer, the system may say that the answer may be incomplete because of access limitations.

It must not:
- imply full coverage when context is partial
- hallucinate around the missing gap
- reveal restricted object existence unless policy allows that metadata visibility

---

## 8. Ranking principles

Retrieval quality is not just semantic similarity.
The platform should rank results using multiple trust-aware signals.

### 8.1 Relevance signals

Useful signals include:
- keyword match
- semantic similarity
- entity title match
- relation proximity
- domain match
- recentness
- source quality
- citation density
- object completeness

### 8.2 Trust-aware ranking signals

Useful trust signals include:
- canonical objects preferred over weak derived artifacts where appropriate
- approved objects preferred over unreviewed drafts
- confirmed decisions preferred over proposed decisions
- fresh content preferred over stale content
- authoritative mirrored references preferred over low-confidence summaries when question intent suggests factual lookup

### 8.3 Ranking rule

The highest-ranked answer candidate should not simply be the most semantically similar text.
It should be the most useful and trustworthy permitted evidence set.

---

## 9. Freshness-aware retrieval

Freshness is essential in knowledge systems.

### 9.1 Freshness signals

Relevant signals:
- last reviewed date
- review_due_at
- freshness_status
- lifecycle state
- superseded status
- recent source updates
- recent confirming activity

### 9.2 Retrieval behavior for stale content

The system should:
- allow retrieval of stale content when relevant and permitted
- visibly label stale content
- prefer fresher equivalents when available
- avoid presenting stale content as current truth

### 9.3 Superseded content

Superseded content should:
- remain historically retrievable when useful
- not rank as default current truth if a newer valid object exists
- clearly indicate what superseded it where possible

---

## 10. Relation-aware retrieval

The platform should use explicit graph-like links to improve discovery and context reconstruction.

### 10.1 Relation expansion examples

Examples:
- from a Meeting to Decisions created in that meeting
- from a Decision to affected Projects
- from an Incident to learned SOP changes
- from a Policy to linked Process and Template objects

### 10.2 Access rule for relations

Every related object must be permission-checked separately.

### 10.3 UX rule

If relation expansion is used in an answer, the answer should make visible which supporting entities were actually used.

---

## 11. Hybrid retrieval architecture

Hybrid retrieval should be the main default for high-value Q&A and synthesis.

### 11.1 Why hybrid retrieval

Pure keyword retrieval misses paraphrases.
Pure semantic retrieval often overreaches and can ignore exact authority signals.

Hybrid retrieval balances:
- precision
- recall
- trustworthiness
- structured filtering
- authority signals

### 11.2 High-level hybrid flow

1. compute allowed scope
2. apply structured facets (domain, type, lifecycle, freshness, approval as applicable) within that scope
3. run exact / keyword retrieval within the facet-bounded scope
4. run semantic retrieval within the same scope (or merge with keyword results)
5. optionally expand with relation-aware candidates (re-check permission per hop; §10.2)
6. merge candidate set
7. rerank with trust-aware signals
8. optionally apply a **second-stage ranker** (e.g. cross-encoder or LLM-based rerank) only when product policy, privacy gateway, and output policy allow sending candidate excerpts to a model; feature-flagged and auditable
9. select final evidence set
10. assemble answer or search result view

### 11.3 Important rule

Hybrid retrieval should be explainable enough that the team can debug poor answers and ranking failures.

### 11.4 Optional second-stage reranking

A second-stage ranker may reorder top-*k* candidates after hybrid merge. It must:
- run only on candidates **already** permitted for the principal
- respect **AI privacy / sanitization** and **output policy** (no bypass of rehydration or placeholder rules; see [AI_PRIVACY_FLOW.md](./AI_PRIVACY_FLOW.md))
- be **disabled or degraded** when policy forbids sending candidate text to external models
- leave **answer traces** sufficient to explain ordering changes for debugging

Do not treat external published benchmark tables as proof of ranking quality on this platform’s data.

---

## 12. Scoped Q&A model

Scoped Q&A is one of the main user-facing AI capabilities in v1.

### 12.1 What scoped Q&A means

The user asks a question in natural language.
The system answers only from permitted context and attaches evidence.

### 12.2 Intended use cases

Examples:
- “What did we decide about the finance operations reporting process?”
- “What are the unresolved blockers from last week?”
- “What changed in planning?”
- “What is the latest approved SOP for onboarding?”
- “What risks were mentioned in the leadership sync?”

### 12.3 Q&A answer expectations

A strong answer should include:
- concise answer
- supporting citations or linked entities
- trust indicators
- owner or authoritative object where relevant
- stale or partial-view messaging where needed

### 12.4 Q&A constraints

Scoped Q&A must not:
- invent authority where none exists
- combine unrestricted private context with visible context
- hide that an answer is based on derived artifacts
- present stale content as confirmed current truth

### 12.5 AI privacy gateway (implementation)

Before any LLM completion for Ask (entity-scoped or global), the API runs the **privacy gateway** (`internal/ai/privacy`): resolve policy → structured/pattern/optional-NER detection → sanitize/tokenize → optional encrypted vault write → model call → governed rehydration. Evidence **ENTITY** header lines skip UUID pattern detection so citation targets remain stable.

Answer traces store `privacy_json` alongside retrieval metrics. See [AI_PRIVACY_POLICY.md](./AI_PRIVACY_POLICY.md), [AI_SANITIZATION_LAYER.md](./AI_SANITIZATION_LAYER.md), [AI_REHYDRATION_LAYER.md](./AI_REHYDRATION_LAYER.md), and [AI_PRIVACY_FLOW.md](./AI_PRIVACY_FLOW.md).

---

## 13. Citation model

Citations are required for trust.

### 13.1 Why citations matter

Citations help users:
- inspect evidence
- validate claims
- navigate to source context
- distinguish grounded answers from generic summaries

### 13.2 Citation targets

A citation may point to:
- canonical entity
- mirrored entity
- derived artifact
- meeting record
- chunk
- source-backed evidence snippet
- source-linked summary input

### 13.3 Citation behavior rule

Citations should point users toward the most meaningful inspectable object, not just the lowest-level chunk.

### 13.4 Citation minimum standard

Important AI answers should include:
- at least one supporting citation when evidence exists
- multiple citations when the answer synthesizes across inputs
- links to authoritative or reviewable objects where possible

### 13.5 Citation UX rule

Citations should help the user move from answer -> supporting entity -> provenance trail.

---

## 14. Supporting entities model

Answers should often include not just citations, but supporting governed objects.

### 14.1 Supporting entity examples

Examples:
- linked Decision objects
- linked Meeting artifacts
- linked Project objects
- linked SOP or Policy objects
- linked Insight artifacts

### 14.2 Why supporting entities matter

They allow users to:
- inspect current truth
- navigate related context
- understand whether the answer is derived from stronger objects
- move beyond a one-shot answer surface

### 14.3 Important rule

Supporting entities should preserve:
- truth mode
- lifecycle state
- freshness state
- approval state where relevant

---

## 15. Answer trace model

Every significant AI answer or synthesis should produce a trace.

### 15.1 Why answer traces matter

They support:
- auditability
- debugging
- trust review
- model evaluation
- governance review
- incident investigation

### 15.2 Trace contents

An answer trace should preserve:
- requester identity
- request type
- request timestamp
- resolved scope snapshot
- retrieval inputs used
- ranked evidence set
- model/task config version
- citations emitted
- final answer reference
- warnings or safety notes

### 15.3 Important rule

Traceability should exist even if the user-facing answer surface stays clean and concise.

---

## 16. Trust indicators

Trust is a product feature, not just a backend property.

### 16.1 Users should be able to see signals such as

- Approved
- In Review
- Draft
- Confirmed Decision
- Superseded
- Stale
- Canonical
- Mirrored
- Derived
- Restricted Scope
- Partial View

### 16.2 Why trust indicators matter

They help users interpret what they are reading.
Without them, all content looks equally authoritative.

### 16.3 Trust UI rule

Trust indicators should be lightweight but unavoidable.
The system should not bury them in deep metadata panels only.

---

## 17. AI task types in v1

The AI layer should support a focused set of tasks.

### 17.1 Summarization
Generate concise synthesis from bounded inputs.

### 17.2 Entity extraction
Extract structured candidates such as Decision or Insight objects.

### 17.3 Decision extraction
Identify likely decisions, rationale, and impacts.

### 17.4 Action item extraction
Extract operational next steps and commitments.

### 17.5 Suggested links
Propose related entities or relation candidates.

### 17.6 Duplicate detection assistance
Suggest likely duplicate or overlapping knowledge artifacts.

### 17.7 Stale detection assistance
Suggest stale or drifted materials that need review.

### 17.8 Scoped Q&A
Answer natural-language questions from allowed evidence only.

### 17.9 Draft editing suggestions
`POST /ai/draft-suggestions` proposes section headings or copy improvements for **draft** entities only. The handler must succeed on `view` + `edit` for the entity, load content **after** those checks, and return JSON for manual apply in the UI. Responses are audited; the API does not persist suggestions as published knowledge.

---

## 18. AI boundaries

These boundaries must be hard, not aspirational.

### 18.1 AI may
- summarize
- synthesize
- extract
- suggest
- rank
- propose links
- help classify
- assist review workflows

### 18.2 AI may not
- bypass permissions
- silently use disallowed context
- define final truth by default
- auto-publish critical artifacts where review is required
- hide absence of evidence
- pretend a derived answer is canonical policy
- overwrite canonical artifacts without workflow

### 18.3 Safe default

When uncertain, the system should classify AI outputs as derived artifacts and route them to review.

---

## 19. Context assembly rules

Context assembly is where many systems quietly become unsafe.

### 19.1 Required context assembly flow

1. identify task
2. identify principal
3. resolve allowed scope
4. retrieve evidence candidates within allowed scope
5. rerank and select evidence
6. assemble structured context with provenance refs
7. invoke model
8. validate output shape
9. attach citations and trust metadata
10. store trace

### 19.2 Context size rule

The system should prefer smaller, better-evidenced context over bloated context windows.

### 19.3 Context composition rule

Context should be intentionally structured where possible:
- canonical objects first when relevant
- supporting derived artifacts second
- recent confirming evidence where needed
- stale or superseded evidence labeled appropriately

### 19.4 Principal-scoped context packs (agent bootstrap)

Some integrations (e.g. future MCP or CLI assistants) may need a **compact bootstrap** of org context before a task starts—analogous in ergonomics to “wake-up” memory in local tooling, but **governed** here.

Requirements:
- **Same permission checks as Ask / search:** the pack is assembled only from objects the principal may read; no broad retrieval with post-filtering.
- **Truth visibility:** items must carry truth mode, lifecycle, and freshness signals; derived content must not present as canonical.
- **No lossy compression as authority:** packs may use terse formatting for tokens, but **must not replace** durable stored evidence; do not use lossy abbreviation of facts as a system of record.
- **TTL and size bounds:** packs should be short-lived hints with explicit max size; refresh when scope or principal changes.
- **Privacy:** respect sanitization, placeholder, and rehydration rules ([AI_REHYDRATION_LAYER.md](./AI_REHYDRATION_LAYER.md)); no raw PII beyond what policy allows for that principal.
- **Optional future surface:** read-only tools (`search`, `entity_get`, `related`) should mirror API semantics and emit audit events when implemented ([API_SURFACE_V1.md](./API_SURFACE_V1.md)).

---

## 20. AI-generated artifact routing

When AI creates something reusable, it must be governed.

### 20.1 Required output metadata

Every reusable AI-generated artifact should receive:
- owner
- domain
- sensitivity
- truth mode
- lifecycle state
- policy source
- provenance
- creation path
- review requirement if applicable

### 20.2 Publication routing options

Outputs may route to:
- draft
- review queue
- approval queue
- non-critical published helper artifact
- monitoring surface

### 20.3 Important rule

AI-generated artifacts must not appear in the system as anonymous, unreviewed “helpful text” with unclear status.

---

## 21. Retrieval for governance workflows

Retrieval is not only for end-user Q&A.

### 21.1 Governance retrieval use cases

Examples:
- find stale policies
- find unowned artifacts
- review extracted decisions
- inspect artifacts created from one source feed
- inspect outputs missing citations
- inspect high-sensitivity derived artifacts

### 21.2 Governance retrieval rule

Governance workflows should support powerful filtering and explainability, not just semantic search.

---

## 22. Search result visibility policy

The system should be conservative about metadata leakage.

### 22.1 Possible visibility levels

For restricted objects, possible policies are:
- invisible
- metadata visible, content hidden
- visible only through explicit grant

### 22.2 v1 default recommendation

Prefer conservative behavior:
restricted objects should usually be invisible unless the system explicitly supports metadata-only visibility for that case.

### 22.3 Reason

Even object existence may leak sensitive information.

---

## 23. AI quality and trust evaluation

The system needs quality signals beyond “the answer sounded good.”

### 23.1 Useful evaluation dimensions

- citation completeness
- citation correctness
- retrieval relevance
- answer usefulness
- trust interpretation clarity
- stale-content handling quality
- partial-answer honesty
- extraction precision
- duplicate suggestion quality
- false authority rate

### 23.2 One especially important metric

Track how often users click from an answer into supporting entities.
This is a strong signal of trust-oriented usage.

### 23.3 Internal evaluation harnesses and external benchmarks

**Internal (recommended):**
- Maintain **offline or staging** evaluation using **synthetic or redacted** corpora that mirror real entity types, domains, and permission boundaries.
- Measure: citation coverage, permission denials, empty-scope rate, stale handling, canonical-vs-derived ranking behavior, and regression when changing retrieval order (e.g. facet-first vs flat semantic).
- Prefer reproducible fixtures checked into the repo or CI-adjacent jobs that do not require production data.

**External benchmarks (caution):**
- Public conversational-memory benchmarks (e.g. LongMemEval-style tasks) measure a **different problem** than multi-tenant governed retrieval: they typically omit org ACLs, retention, connector scope, and audit requirements.
- **Do not** cite third-party leaderboard scores as product SLAs or security arguments.
- Third-party tooling that reports very high recall on raw verbatim storage is informative for **design tradeoffs** (preserving evidence vs summarizing early) but does not override ADR-0006 guardrails or permission models.

---

## 24. Failure modes and safe handling

This layer will fail in subtle ways if not designed honestly.

### 24.1 Common failure modes

- answer is semantically plausible but weakly grounded
- retrieval finds stale content first
- derived artifacts outrank stronger authoritative objects
- relation expansion surfaces noisy context
- citations point to low-value chunks instead of meaningful objects
- partial answers sound complete
- model output overstates confidence
- access scoping is too narrow and answer quality drops
- access scoping is too broad and becomes unsafe

### 24.2 Safe handling expectations

The system should:
- prefer evidence over fluency
- make stale or derived status visible
- surface when confidence is limited
- preserve user path to inspect sources
- keep traces for debugging
- avoid pretending uncertainty does not exist

---

## 25. Review requirements for AI-generated materials

Some AI-generated outputs should require review before broad use.

### 25.1 Strong review candidates

Examples:
- extracted decisions that may shape future work
- planning summaries used for execution alignment
- SOP drafts
- policy drafts
- sensitive cross-domain summaries
- leadership summaries

### 25.2 Lighter-weight outputs

Some low-risk outputs may publish more easily:
- private draft notes
- low-sensitivity helper summaries
- duplicate suggestions
- stale-content alerts
- reviewer-only suggestions

### 25.3 Key principle

The higher the authority risk, the stronger the review requirement.

---

## 26. Observability requirements

AI and retrieval behavior must be inspectable.

### 26.1 Retrieval observability

Track:
- query volume
- retrieval latency
- evidence set size
- semantic vs keyword contribution
- ranking shifts
- access-filtered candidate counts
- stale-result frequency

### 26.2 AI observability

Track:
- answer generation count
- model latency
- citation coverage
- output type distribution
- review routing rates
- failure rates
- partial-answer rates

### 26.3 Governance observability

Track:
- answers without citations
- sensitive outputs without required review
- artifacts with missing provenance
- users frequently hitting restricted-scope boundaries
- stale artifacts used in answers

---

## 27. UX requirements

### 27.1 Search UI should support
- keyword search
- filters
- entity-type narrowing
- domain filtering where allowed
- trust indicators in results
- easy drill-down to entity details

### 27.2 Q&A UI should support
- concise answer
- supporting citations
- supporting entities
- trust badges
- stale / superseded messaging
- partial-view messaging where needed

### 27.3 Entity detail UI should support
- title
- owner
- truth mode
- lifecycle state
- freshness
- provenance
- relations
- version history
- supporting source trail

### 27.4 Important UX rule

The UI should make the system feel trustworthy by design, not only powerful.

---

## 28. Testing requirements

This layer requires strong testing because failures are subtle.

### 28.1 Mandatory test areas

- permission-aware retrieval scoping
- relation expansion permission checks
- stale ranking behavior
- canonical vs derived ranking behavior
- citation attachment correctness
- answer trace creation
- partial-answer handling
- metadata leakage prevention
- AI context assembly scope enforcement
- review routing for governed generated artifacts

### 28.2 Example scenarios

Examples:
- user can view a decision but not linked confidential incident
- stale SOP is retrieved but clearly labeled and outranked by active version
- extracted summary is cited as derived, not canonical
- canonical policy outranks weak semantically similar meeting note
- partial answer is returned when restricted domain content is missing
- AI output requiring review does not auto-publish

---

## 29. Anti-patterns to avoid

Do not:
- build a generic unrestricted AI chat over all company data
- retrieve first and permission-filter later
- optimize answer style over evidence quality
- let derived artifacts look canonical
- hide stale state
- use citations as decoration only
- treat vector similarity as truth
- let AI outputs bypass workflow
- bury trust indicators in metadata corners
- let low-authority summaries outrank approved canonical objects by default

---

## 30. Open questions

- How much metadata-only visibility should ever be allowed for restricted objects?
- Which trust badges should be shown inline versus in detail views?
- How aggressive should freshness-aware reranking be?
- Should some Q&A modes prefer canonical-only evidence for certain question types?
- When should the system answer versus decline due to insufficient evidence?
- How should user feedback on weak answers route into retrieval tuning?

---

## 31. Final stance

This system should not win by sounding smart.
It should win by being trustworthy.

A strong answer should let the user understand:
- what the answer says
- what supports it
- what kind of truth it represents
- how current it is
- what may be missing
- where to inspect further

If AI feels magical but untraceable, the system is weak.
If AI feels bounded, evidence-backed, and operationally trustworthy, the system is strong.