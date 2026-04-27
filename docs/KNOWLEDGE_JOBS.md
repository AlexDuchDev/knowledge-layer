# KNOWLEDGE_JOBS.md

## 1. Purpose

This document defines the Knowledge Operations and Knowledge Jobs model for the Organizational Memory & Knowledge Operations Platform.

It establishes:
- what a knowledge job is
- why knowledge jobs exist
- job categories and supported task types
- job definition structure
- trigger modes
- execution model
- input scope rules
- output routing and publication rules
- review and approval interactions
- access and operator permissions
- observability and failure handling
- v1 example jobs
- product constraints for safe and governed operation

Knowledge jobs are one of the core differentiators of the platform.
They turn governed memory into governed operations over knowledge.

**Implementation map (validators, orchestrator, HTTP gates, migrations):** [knowledge-jobs-engine.md](./knowledge-jobs-engine.md).

**Job Builder (control plane APIs, presets, admin UI):** [job-builder.md](./job-builder.md).

---

## 2. Why knowledge jobs matter

In most companies, important knowledge work is repetitive but poorly operationalized.

Examples:
- every week someone reconstructs what happened across chats and meetings
- after planning, someone manually summarizes commitments and risks
- after incidents, someone tries to extract learnings and follow-up process changes
- leaders repeatedly ask for status digests built from scattered sources
- teams manually reassemble context that the system should already understand

Knowledge jobs exist to systematize this work.

They allow the platform to:
- transform fragmented inputs into structured outputs
- operate on bounded source scopes
- run repeatably and visibly
- enforce review and publication rules
- preserve provenance and execution traces
- create governed artifacts instead of disposable AI outputs

Without knowledge jobs, the system is a memory layer.
With knowledge jobs, it becomes a knowledge operations engine.

---

## 3. Core principles

### 3.1 Jobs are explicit system objects
A knowledge job is not an invisible background prompt or an ad hoc AI action.
It is a first-class governed object with configuration, owner, scope, trigger, outputs, and traceability.

### 3.2 Jobs are process-bound
A job should exist because it supports a real operating process, not because the model can generate text.

### 3.3 Jobs operate on controlled source scope
Every job must run against explicitly allowed sources or source classes.
No job should implicitly read “everything available.”

### 3.4 Jobs produce governed outputs
Outputs must receive domain, sensitivity, provenance, lifecycle state, and policy context.

### 3.5 Jobs do not define truth automatically
Derived outputs may become useful artifacts, but critical or authoritative materials must follow review and approval rules where required.

### 3.6 Jobs must be inspectable
An operator or admin should be able to answer:
- what this job does
- why it exists
- which inputs it used
- when it ran
- what it produced
- who reviewed the outputs
- what failed

### 3.7 Jobs should be repeatable
The same job definition should be reusable over time, with stable semantics and clear trigger behavior.

### 3.8 AI is a tool inside jobs, not the job’s authority
AI may help summarize, extract, classify, suggest links, or transform content.
It does not bypass access, provenance, or workflow rules.

---

## 4. What a knowledge job is

A knowledge job is a governed, repeatable operation that runs over a bounded set of source inputs or entities and produces structured, traceable outputs.

A job definition includes:
- purpose
- owner
- source scope
- trigger
- operator permissions
- output policy
- review requirements
- publication mode
- execution settings

A job run includes:
- resolved input set
- execution status
- warnings and errors
- output artifacts
- trace and provenance metadata

---

## 5. Job categories

The system should support a small, clear taxonomy in v1.

### 5.1 Summarization jobs

Purpose:
Produce condensed, structured summaries from governed source inputs.

Examples:
- weekly team digest
- planning summary
- leadership brief
- meeting recap
- domain update summary

Typical outputs:
- summary artifact
- linked entities
- optional review task

---

### 5.2 Extraction jobs

Purpose:
Extract structured objects or signals from raw or normalized inputs.

Examples:
- decision extraction
- action item extraction
- blocker extraction
- risk extraction
- insight extraction

Typical outputs:
- candidate Decision entities
- draft Insight entities
- extracted action item sets
- reviewable derived artifacts

---

### 5.3 Consolidation jobs

Purpose:
Merge and synthesize related information across multiple inputs or source windows.

Examples:
- planning consolidation across transcript + Jira + notes
- project state consolidation across Slack + Jira + meetings
- policy comparison and consolidation draft

Typical outputs:
- consolidated summary
- structured cross-source artifact
- linked canonical candidates

---

### 5.4 Monitoring jobs

Purpose:
Detect important changes, emerging risks, drift, or notable patterns over time.

Examples:
- stale content detection
- duplicate knowledge detection
- repeated blocker monitoring
- incident-pattern monitoring
- missing owner monitoring

Typical outputs:
- alerts
- flagged entities
- review queue items
- monitoring reports

---

### 5.5 Transformation jobs

Purpose:
Convert one representation into another governed representation.

Examples:
- transcript to meeting object
- source notes to SOP draft
- fragmented decisions to canonical decision draft
- imported reference material to structured knowledge draft

Typical outputs:
- transformed draft entities
- structured artifacts ready for review
- linked provenance-preserving outputs

---

### 5.6 Publishing jobs

Purpose:
Route approved or prepared artifacts into the right surfaced destination or lifecycle state.

Examples:
- publish reviewed weekly digest
- move approved SOP to active state
- publish approved domain summary
- route reviewed policy update into canonical set

Typical outputs:
- state transition
- publication record
- notifications
- audit events

---

## 6. What belongs in a job versus outside a job

### 6.1 Belongs inside a job

A task belongs inside a knowledge job when it is:
- repeatable
- process-bound
- governed
- dependent on controlled input scope
- valuable as a traceable output
- likely to be scheduled, event-triggered, or operator-triggered

### 6.2 Does not belong as a first-class job

A task may not need a job when it is:
- a one-off lightweight search
- a simple retrieval request
- a local UI convenience transformation
- a transient formatting step
- a prompt experiment with no operational meaning

### 6.3 Useful decision rule

If the result should be:
- reviewed
- versioned
- audited
- routed
- repeated
- owned

then it probably should be a job.

---

### 6.1 AI privacy (jobs)

Processors that invoke an LLM must use `internal/ai/privacy.PrivacyGateway` with `job_type`, `output_type`, and `job_run_id` in `PolicyContext` / `InvokeInput`. The current `weekly_digest` path does not call an LLM; see [AI_PRIVACY_FLOW.md](./AI_PRIVACY_FLOW.md).

## 7. Knowledge job object model

### 7.1 KnowledgeJob

A reusable job definition.

Recommended fields:
- `id`
- `name`
- `job_type`
- `purpose`
- `description`
- `owner_id`
- `operator_scope`
- `source_scope`
- `trigger_type`
- `output_type`
- `output_domain_id`
- `output_sensitivity_level`
- `publication_mode`
- `review_required`
- `approval_required`
- `sanitization_rules`
- `config_json`
- `status`
- `created_at`
- `updated_at`

### 7.2 Suggested job types

- `summarization`
- `extraction`
- `consolidation`
- `monitoring`
- `transformation`
- `publishing`

### 7.3 Suggested job statuses

- `draft`
- `active`
- `paused`
- `archived`

### 7.4 Why a strong job object matters

A job should be understandable even before execution.
An operator should not need to inspect code or prompt internals to know what the job is supposed to do.

---

## 8. Job trigger model

A job may run through multiple trigger patterns.

### 8.1 Supported trigger types

- `manual`
- `scheduled`
- `event_driven`
- `window_based`
- `conditional`

### 8.2 Manual trigger

Started directly by an authorized human.

Use when:
- an operator wants control over timing
- the job is run occasionally
- review of input selection matters
- the workflow is not yet stable enough for automation

### 8.3 Scheduled trigger

Runs on a recurring cadence.

Use when:
- weekly or daily digests are needed
- stale checks should run periodically
- routine reporting or monitoring is expected

### 8.4 Event-driven trigger

Runs when a defined event occurs.

Use when:
- a meeting transcript arrives
- a source feed ingests a matching artifact
- a workflow state changes
- a new high-priority incident is detected

### 8.5 Window-based trigger

Runs against a bounded time or input window.

Use when:
- weekly summaries need Monday-to-Friday scope
- planning summaries need last planning session inputs
- meeting clusters should be grouped by time range

### 8.6 Conditional trigger

Runs when a system condition is satisfied.

Use when:
- enough new inputs exist
- a threshold of change has been reached
- stale content crosses a risk boundary
- review backlog exceeds a limit

---

## 9. Job definition model

A job definition must contain enough structure to be operated and governed safely.

### 9.1 Required definition fields

Every job should define:
- business purpose
- owner
- job type
- source scope
- trigger configuration
- allowed operators
- output route
- output domain
- output sensitivity
- publication mode
- review requirement
- status

### 9.2 Strongly recommended fields

Also include:
- narrative of intended use
- expected output structure
- fallback behavior
- retry policy
- confidence handling expectations
- maximum input window or size constraints
- notification behavior
- observability tags

### 9.3 Job purpose standard

The purpose field should describe:
- why this job exists
- what business or operating workflow it supports
- what useful output it should produce
- what human action it enables downstream

Bad:
“Summarize chat”

Better:
“Produce a weekly Finance Ops digest from the approved Telegram finance feed, highlighting decisions, blockers, risks, and unresolved questions for reviewer validation before publication.”

---

## 10. Source scope model

Source scope is one of the most important parts of any job.

### 10.1 Source scope definition

A source scope specifies which inputs a job is allowed to use.

A source scope may include:
- explicit source feed IDs
- domain-limited source classes
- source type filters
- time window constraints
- entity-type inputs
- upstream artifact status filters

### 10.2 Source scope rules

A job must not:
- implicitly expand to all accessible company content
- use inputs outside configured and permitted source scope
- cross into unrelated domains unless explicitly allowed
- bypass source-level allowed job restrictions

### 10.3 Source scope examples

#### Example 1: Weekly finance digest
- source feeds: `telegram://finance_ops_internal`, `slack://finance-daily`
- time window: last 7 days
- allowed source types: messages, transcript summaries
- output domain: Finance

#### Example 2: Planning summary
- source feeds: planning transcript source, Jira board snapshot, planning notes doc
- input window: latest planning cycle
- output domain: Product

### 10.4 Input resolution snapshot

At run time, the system should store the resolved input scope snapshot so that the run is explainable later.

---

## 11. Operator permissions model

Not every user should be able to run every job.

### 11.1 Job permissions should distinguish

- create job
- edit job
- activate or pause job
- run job manually
- inspect run details
- cancel run
- review outputs
- approve outputs
- publish outputs

### 11.2 Operator scope

A job should define its allowed operators via:
- specific users
- roles
- team scope
- domain-level operator permissions

### 11.3 Important rule

A user may be allowed to view a job but not run it.
A user may be allowed to run a job but not publish its outputs.
A user may be allowed to review outputs but not edit the job definition.

### 11.4 Scheduled job execution rule

When a job runs on schedule, it must run under stored governed permissions and scope.
It must not inherit universal worker privileges.

---

## 12. Execution model

A job run is a first-class execution record.

### 12.1 JobRun

Recommended fields:
- `id`
- `knowledge_job_id`
- `initiated_by_type`
- `initiated_by_id`
- `trigger_type`
- `status`
- `input_scope_snapshot`
- `started_at`
- `completed_at`
- `warning_count`
- `error_count`
- `trace_ref`
- `execution_metrics_json`

### 12.2 Suggested run statuses

- `queued`
- `running`
- `completed`
- `failed`
- `cancelled`
- `partial_success`

### 12.3 Execution stages

A typical job run should include:
1. permission and trigger validation
2. input scope resolution
3. input retrieval
4. preprocessing or grouping
5. AI or rule-based execution
6. post-processing and validation
7. output creation
8. routing to review or publication path
9. audit and notification emission

### 12.4 Idempotency expectation

Where possible, jobs should be designed to avoid duplicate output creation from accidental reruns.
At minimum, duplicate publication should be guarded.

---

## 13. Job execution pipeline

### 13.1 Stage 1 — Start validation

Before execution, the system should validate:
- job is active
- trigger is allowed
- actor is allowed
- source feeds are still valid
- source feeds still allow this job type
- output policy remains valid

### 13.2 Stage 2 — Resolve inputs

The system should:
- resolve source scope
- resolve time or event window
- apply domain and access constraints
- fetch only allowed input records
- store input snapshot for traceability

### 13.3 Stage 3 — Process inputs

The system may:
- group inputs
- de-duplicate within run
- prepare chunks
- assemble structured task context
- call model or deterministic logic

### 13.4 Stage 4 — Validate outputs

The system should:
- validate output structure
- attach provenance
- classify output truth mode
- assign owner, domain, sensitivity, and policy source
- apply confidence or quality checks where relevant

### 13.5 Stage 5 — Route outputs

Outputs may go to:
- draft state
- review queue
- approval flow
- published non-critical state
- monitoring alert surface

### 13.6 Stage 6 — Record trace

The system should store:
- input refs
- model/task config version
- output refs
- warnings
- errors
- timing
- audit events

---

## 14. Output model

Job outputs are not just strings.
They are governed artifacts or transitions.

### 14.1 Output types

Common output types in v1:
- summary artifact
- candidate Decision entity
- draft Insight entity
- meeting summary artifact
- monitoring alert
- review task
- publication transition
- link suggestions

### 14.2 JobOutput object

Recommended fields:
- `id`
- `job_run_id`
- `output_type`
- `target_entity_id`
- `target_entity_type`
- `review_task_id`
- `publication_status`
- `created_at`

### 14.3 Output rules

Every output should receive:
- owner
- domain
- sensitivity
- truth mode
- provenance
- workflow state
- policy source

### 14.4 Important rule

A job output must never enter the system as an unowned, policy-free, contextless blob.

---

## 15. Publication modes

Publication mode defines what happens after the output is created.

### 15.1 Suggested publication modes

- `draft_only`
- `review_required`
- `approval_required`
- `publish_if_safe`
- `state_transition_only`

### 15.2 Draft only

Outputs are created as drafts and require human follow-up.

Use when:
- confidence is low
- authority should remain human-controlled
- the workflow is exploratory

### 15.3 Review required

Outputs enter a review queue before becoming visible as governed outputs.

Use when:
- summaries influence decisions
- extracted entities need confirmation
- source quality is uneven
- trust matters more than speed

### 15.4 Approval required

Outputs require formal approval before becoming active or authoritative.

Use when:
- policy-like content is generated
- SOP drafts become active procedures
- cross-domain artifacts need signoff

### 15.5 Publish if safe

Outputs may publish automatically under narrow and explicit conditions.

Use carefully for:
- low-risk monitoring signals
- non-sensitive routine summaries
- helper artifacts with low authority risk

### 15.6 State transition only

The job does not create new content; it changes or routes existing governed artifacts.

Example:
- publish approved digest
- archive stale outputs
- move approved policy to active

---

## 16. Review and approval interaction

Knowledge jobs and governance must work together tightly.

### 16.1 Review requirement

A job may require review for:
- all outputs
- outputs above certain sensitivity
- outputs of certain types
- outputs below a confidence threshold
- outputs touching critical domains

### 16.2 Approval requirement

A job may require approval when:
- it affects canonical policy or SOP state
- it creates high-authority artifacts
- it publishes sensitive content broadly

### 16.3 Reviewer assignment

Reviewer assignment may come from:
- job configuration
- domain owner defaults
- output entity type rules
- approval flow rules

### 16.4 Review UX expectation

A reviewer should see:
- generated content
- supporting citations
- input provenance
- output classification
- owner
- domain
- sensitivity
- current workflow state

### 16.5 Important rule

If the output matters enough to influence company behavior, the review surface must make it easy to inspect the evidence.

---

## 17. Truth modes and jobs

Jobs must respect the system’s truth classification model.

### 17.1 Canonical in platform

A job may contribute to canonical objects only through controlled workflows.

Examples:
- creating a draft SOP that later becomes active
- generating an update suggestion to a canonical handbook

### 17.2 Mirrored authority

Jobs may read mirrored inputs and produce derived outputs, but should not pretend to own the mirrored source’s truth.

Example:
- a Jira-based project summary is derived, not authoritative over Jira itself

### 17.3 Derived artifact

Most job outputs in v1 should default to `derived_artifact` unless there is an explicit workflow that transitions them into something more authoritative.

### 17.4 Safe default

When unsure, outputs should be classified as derived artifacts and routed to review.

---

## 18. Provenance requirements

Provenance is mandatory for job outputs.

### 18.1 Every job run should preserve

- job definition used
- trigger used
- input scope snapshot
- source refs
- raw artifact refs where applicable
- normalized record refs where applicable
- output refs
- actor or execution identity

### 18.2 Every output should preserve

- job run ID
- source inputs
- creation path
- reviewer or approver if applicable
- timestamps
- relevant model/task config version where AI is involved

### 18.3 Why this matters

A user or admin should be able to inspect an output and understand:
- why it exists
- what it was based on
- whether it was reviewed
- how much to trust it

---

## 19. Access and jobs

Jobs must obey the access model strictly.

### 19.1 Job creation access

Only authorized users should create jobs.

### 19.2 Job execution access

A job run must respect:
- allowed operators
- source feed restrictions
- domain restrictions
- sensitivity boundaries
- review and publication constraints

### 19.3 Job outputs and access inheritance

Outputs should inherit from:
- explicit job output policy
- otherwise source feed policy
- otherwise domain default policy

### 19.4 AI inside jobs

If AI is used inside a job:
- only allowed inputs may be assembled
- only allowed outputs may be created
- no hidden broad context may be injected
- trace and citation support must remain available

### 19.5 Cross-domain jobs

Cross-domain jobs should be rare and explicit.
They must define:
- allowed domains
- operator permissions
- output domain or multi-domain handling
- stricter review expectations where needed

---

## 20. Scheduling model

### 20.1 Scheduling support in v1

The platform should support:
- recurring schedules
- one-off scheduled runs
- manual runs
- event-based triggers
- time window resolution

### 20.2 Window handling

Window-based jobs should define:
- time range
- inclusion rules
- late-arriving input behavior
- deduplication behavior for overlapping windows

### 20.3 Schedule safety

The system should guard against:
- overlapping duplicate runs
- runaway retries
- stale schedules on archived jobs
- hidden schedule drift after config changes

### 20.4 Schedule visibility

Operators should be able to see:
- next planned run
- last successful run
- last failed run
- recent execution history
- active trigger configuration

---

## 21. Monitoring and alerting jobs

Monitoring jobs deserve special handling because they often produce signals, not polished artifacts.

### 21.1 Monitoring job outputs may include

- stale-content flags
- duplicate-content suggestions
- repeated-risk alerts
- owner-missing alerts
- policy exception flags

### 21.2 Monitoring job publication rule

Monitoring outputs should usually route to:
- admin surfaces
- governance queues
- alerts
- review lists

not directly to authoritative knowledge objects.

### 21.3 Signal quality rule

Monitoring jobs should preserve enough evidence for a human to inspect why the flag exists.

---

## 22. Failure handling

Jobs will fail. The system should treat that as normal operational reality.

### 22.1 Error classes

Suggested job error classes:
- configuration error
- permission error
- source scope resolution error
- input retrieval error
- model execution error
- output validation error
- routing error
- notification error

### 22.2 Retry policy

Retry may be appropriate for:
- transient retrieval failures
- temporary model errors
- queue issues
- short-lived downstream storage failures

Retry is usually not appropriate for:
- invalid job config
- revoked access
- invalid output schema
- forbidden source scope
- missing required review policy

### 22.3 Partial success

Some jobs may partially succeed.

Example:
- summary generated
- one linked candidate entity failed validation
- review task created successfully

The run should capture partial success rather than flattening to pure failure.

### 22.4 Operator visibility

An operator should be able to inspect:
- what failed
- whether any outputs were produced
- whether retry is safe
- whether manual intervention is needed

---

## 23. Observability

Jobs should be observable as infrastructure, not as mystery AI runs.

### 23.1 Job-level metrics

Useful metrics:
- runs per job
- success rate
- failure rate
- median runtime
- queue time
- review rate
- approval rate
- output count
- publish rate
- rejection rate

### 23.2 Output quality metrics

Useful metrics:
- decision extraction precision
- summary usefulness rating
- review rejection rate
- duplicate output rate
- stale flag acceptance rate
- citation completeness rate

### 23.3 Governance metrics

Useful metrics:
- outputs missing owner
- outputs missing provenance
- outputs awaiting review too long
- outputs auto-published under safe mode
- critical outputs published without required review

### 23.4 Run trace requirements

Each run should make visible:
- trigger
- inputs
- outputs
- duration
- warnings
- errors
- routing result
- downstream review state

---

## 24. UX requirements

### 24.1 Knowledge Jobs UI should support

- list of jobs
- job status
- owner
- trigger type
- next run
- last run
- recent results
- source scope summary
- output policy summary

### 24.2 Job detail page should support

- job purpose
- configuration
- source scope
- allowed operators
- trigger settings
- output route
- review requirement
- run history
- output history
- related review tasks
- audit summary

### 24.3 Run detail page should support

- run status
- trigger details
- inputs used
- output artifacts
- citations or provenance references
- warnings and errors
- retry or rerun actions if authorized

### 24.4 Important UX rule

A knowledge job should feel like an operational process, not like a hidden prompt template.

---

## 25. Example jobs for v1

### 25.1 Weekly daily digest

Purpose:
Summarize the operational week for a team from governed communication sources.

Inputs:
- Telegram daily chat
- Slack daily channel
- Fireflies or Granola transcripts

Operation:
- summarize progress
- extract blockers
- extract risks
- extract decisions
- identify open questions

Output:
- weekly digest artifact
- optional linked Decision candidates
- review task

Publication mode:
- review required

---

### 25.2 Planning summary

Purpose:
Create a structured summary after planning.

Inputs:
- planning transcript
- Jira board snapshot
- planning notes

Operation:
- extract commitments
- extract risks
- extract decisions
- extract open questions

Output:
- planning summary artifact
- linked Decision objects or candidates
- review task for owner

Publication mode:
- review required

---

### 25.3 Decision extraction from meeting transcripts

Purpose:
Identify decision candidates from governed meeting inputs.

Inputs:
- meeting transcripts
- supporting notes

Operation:
- detect decision statements
- cluster rationale
- identify impacted project or domain
- produce candidate Decision entities

Output:
- draft or candidate Decision entities
- reviewer queue

Publication mode:
- review required

---

### 25.4 Split concatenated transcript / chat exports (ingestion hygiene)

Purpose:
Improve chunking, retrieval, and review quality when a source delivers **many sessions in one file**.

Inputs:
- one or more **raw artifacts** flagged as mega-export or connector-specific “bundle” types
- optional parent export retained for audit ([INGESTION_AND_CONNECTORS.md](./INGESTION_AND_CONNECTORS.md) §12.7)

Operation:
- detect session or time boundaries (format-specific)
- emit child raw artifacts (or normalized records) per segment
- link segments to source feed and ingestion run
- trigger re-normalization and downstream indexing as needed

Output:
- additional raw artifacts and normalized records
- observability events for split counts and failures

Publication mode:
- **not** a user-facing publish; structural ingestion only. Downstream extracted entities still follow normal review rules.

---

### 25.5 Stale policy detector

Purpose:
Identify policies that may require freshness review.

Inputs:
- Policy entities
- freshness rules
- review dates
- related activity signals where useful

Operation:
- detect overdue review
- detect possible drift indicators
- flag items for review

Output:
- monitoring alerts
- review tasks

Publication mode:
- alert / review routing

---

### 25.6 Duplicate insight detector

Purpose:
Find likely overlapping insight artifacts.

Inputs:
- Insight entities
- Customer Insight entities
- recent summaries

Operation:
- compare semantic overlap
- suggest duplicate clusters
- propose merge candidates

Output:
- duplicate suggestions
- governance review items

Publication mode:
- draft / review only

---

## 26. Templates versus jobs

The platform should distinguish job templates from job instances.

### 26.1 Job template

A reusable pattern for creating jobs.

Examples:
- weekly digest template
- decision extraction template
- planning summary template

### 26.2 Job instance

A configured operational job for a real domain or source scope.

Example:
“Finance Weekly Digest from finance_ops_internal Telegram chat”

### 26.3 Why this matters

Templates speed setup.
Instances carry real governance, scope, ownership, and policy.

---

## 27. Anti-patterns to avoid

Do not:
- create jobs with vague purpose
- let jobs run over undefined scope
- treat job outputs as disposable AI text
- auto-publish sensitive outputs casually
- hide job behavior in prompts only
- let scheduled jobs inherit broad system access
- skip provenance on job outputs
- let one giant generic job replace clear, process-bound jobs
- create too many job types with overlapping semantics
- optimize for novelty instead of repeatable operational value

---

## 28. Open questions

- Which jobs should be first-class templates in v1?
- Which outputs can safely use `publish_if_safe`?
- Should action items become first-class entities or remain job output structures in v1?
- How much per-job customization is acceptable before jobs become too hard to govern?
- Which domains need the strictest review defaults?
- How much operator control over input selection should exist for manual runs?
- Which monitoring jobs provide the earliest visible value?

---

## 29. Final stance

Knowledge jobs are where the platform stops being passive.

A strong knowledge job system should make it easy to answer:
- what recurring knowledge work exists
- who owns it
- what inputs it uses
- what outputs it creates
- how safe its publication path is
- how trustworthy its results are
- what failed and why

If jobs feel like hidden prompt automation, the system is too weak.
If jobs feel like governed operational processes over knowledge, the system is on the right track.