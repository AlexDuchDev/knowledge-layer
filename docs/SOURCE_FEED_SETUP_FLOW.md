# SOURCE_FEED_SETUP_FLOW.md

## 1. Purpose

This document defines the UX flow for setting up a Source Feed in v1.

It covers:
- the end-to-end setup flow
- required steps and decisions
- validation rules
- preview before activation
- activation readiness
- error handling
- admin guidance and UX behavior

The goal is to make source onboarding:
- self-serve where practical
- governance-first
- understandable
- safe
- not overly technical

This is not a connector implementation spec.
It is the UX and product flow contract for Source Feed setup.

---

## 2. Why this flow matters

Source Feeds are one of the most important control points in the entire system.

A weak source setup flow creates risk:
- unclear ownership
- wrong domain assignment
- weak sensitivity handling
- bad allowed-job configuration
- accidental ingestion of the wrong source
- admin confusion
- poor trust in the system

A strong setup flow should make source connection feel:
- deliberate
- governed
- inspectable
- reversible
- operationally credible

It should not feel like:
- a casual plugin install
- a blind OAuth click
- a hidden technical integration screen
- a dev-only workflow

---

## 3. UX goals

The Source Feed setup flow should help admins:
- choose the right connector
- authenticate cleanly
- map the correct source
- assign the right governance metadata
- preview what will be ingested
- understand activation readiness
- activate safely
- recover from common setup issues

The flow should reduce ambiguity, not just collect fields.

---

## 4. Core product stance

A Source Feed is not just a connection.
It is a governed ingestion boundary.

Therefore the setup flow must explicitly capture:
- connector
- source mapping
- owner
- domain
- sensitivity
- allowed jobs
- ingestion mode
- sync mode
- activation readiness

This is one of the strongest differences between this product and simpler content ingestion tools.

---

## 5. High-level setup stages

Recommended flow:

1. Choose Connector
2. Authenticate Connector
3. Map Source
4. Configure Governance
5. Choose Ingestion Behavior
6. Preview Source
7. Review and Validate
8. Activate

This should feel like a guided sequence, not a flat long form.

---

## 6. Entry points

Admins should be able to start Source Feed setup from:
- Source Feeds list page
- empty state on Source Feeds page
- connector-specific quick action if available later
- rollout/setup kits later

Primary CTA:
- Connect Source Feed
- Add Source Feed

Avoid vague labels like:
- Add integration
- Connect data
- New source

“Source Feed” is the correct product concept.

---

## 7. Step 1 — Choose Connector

## 7.1 Purpose

Choose the external system type the source belongs to.

### Examples
- Telegram
- Slack
- Email
- Fireflies
- Granola
- Jira
- Trello
- Notion
- Google Drive / Docs

---

## 7.2 UI contents

The connector selection screen should show:
- connector name
- short description
- source type category
- readiness/support state if relevant
- maybe a note about typical use

### Example descriptions
- Telegram — governed ingestion from explicitly connected chats
- Slack — governed ingestion from selected channels and threads
- Google Docs — governed ingestion from selected document sources

---

## 7.3 UX rules

- connector choice should feel explicit
- unsupported or limited connectors should be labeled honestly
- v1 constraints should be visible where important

### Example
For Telegram:
- In v1, Telegram is ingestion-only
- Telegram is not a bot interface or output channel

---

## 8. Step 2 — Authenticate Connector

## 8.1 Purpose

Establish credentials or access required to inspect and use the external source.

### Possible auth patterns
- OAuth
- API token
- service credential
- admin-provided external integration secret

The exact auth pattern may differ by connector.

---

## 8.2 UI contents

The auth step should show:
- connector name
- what permissions are being requested at a high level
- current auth status
- reconnect / test connection actions where relevant

### Status states
- Not connected
- Connected
- Invalid credentials
- Needs refresh
- Auth failed

---

## 8.3 UX rules

- auth success should not imply source feed is ready
- auth failure should be clear and actionable
- avoid showing raw technical jargon unless needed

### Example good copy
- Connection established
- You still need to select the source and define governance settings

---

## 8.4 Error handling

If auth fails, the UI should show:
- what failed at a practical level
- what the admin can try next
- whether setup can continue or not

Do not just show raw connector error text unless no better explanation is available.

---

## 9. Step 3 — Map Source

## 9.1 Purpose

Select the actual source boundary inside the external system.

### Examples
- specific Telegram chat
- specific Slack channel
- specific mailbox or label
- specific Notion page tree
- specific Google Drive document or folder
- specific Jira project/board
- specific transcript source

---

## 9.2 UI contents

The source mapping step should support:
- source lookup or selection
- source identifier display
- display name or editable feed name
- source URI/reference preview
- source type metadata where relevant

---

## 9.3 UX rules

- source mapping must make the chosen source legible
- admins should not activate a feed without confidence in what source is selected
- if source ambiguity exists, the UI should ask for clarification

### Important rule
A source must not be “loosely connected.”
It must be explicitly mapped.

---

## 10. Step 4 — Configure Governance

This is the most important step.

## 10.1 Required governance fields

The admin must set:
- owner
- domain
- sensitivity
- allowed jobs

These fields are required before activation.

---

## 10.2 Owner

### Purpose
Assign a human owner accountable for the source feed.

### UX requirements
- choose from users
- show role/team context where useful
- explain why owner matters

### Suggested help text
The owner is responsible for the feed’s governance and operational accountability.

---

## 10.3 Domain

### Purpose
Assign the business knowledge boundary.

### UX requirements
- choose from existing domains
- show domain owner where useful
- make domain assignment feel important, not decorative

### Suggested help text
The domain affects default access, governance, and downstream routing.

---

## 10.4 Sensitivity

### Purpose
Set the default sensitivity posture for content derived from this source.

### Recommended options
- public_internal
- team_restricted
- domain_restricted
- leadership_restricted
- strictly_confidential

### UX requirements
- explain each sensitivity level clearly
- keep wording practical
- do not rely only on internal labels if clearer helper text is needed

---

## 10.5 Allowed jobs

### Purpose
Limit what types of knowledge operations may run on this feed.

### Examples
- weekly digest
- planning summary
- decision extraction
- issue extraction
- stale scan, where relevant

### UX requirements
- allowed jobs should be selectable and visible
- this should feel like a real control point
- if no jobs are allowed, that should be explicit

---

## 10.6 Governance step UX rule

This step should make the admin feel:
- “I am defining how this source is governed”
not
- “I am filling metadata to get through setup”

---

## 11. Step 5 — Choose Ingestion Behavior

## 11.1 Ingestion mode

The admin should choose the intended ingestion behavior.

### Suggested modes
- raw_capture_only
- governed_processing
- governed_processing_with_jobs

### UX rule
The choice should be explained in plain product language.

### Example helper text
- Raw capture only: store source evidence conservatively
- Governed processing: allow indexing and structured downstream use
- Governed processing with jobs: allow approved knowledge operations on this feed

---

## 11.2 Sync mode

The admin should choose or confirm sync behavior where relevant.

### Suggested modes
- full import
- incremental sync
- event-driven

If only one mode is available, the UI should say so instead of pretending this is a meaningful choice.

---

## 12. Step 6 — Preview Source

## 12.1 Purpose

Allow the admin to inspect what this source feed is likely to ingest before activation.

This is one of the most important trust-building steps in the setup flow.

---

## 12.2 Preview should show

A bounded preview of:
- sample source items
- source metadata
- likely content shape
- timestamps
- participant/author presence where available
- title/thread/document structure where available

### Examples
- recent message samples from a chat
- thread/message examples from Slack
- sample document titles from Docs/Notion
- transcript metadata and sample sections

---

## 12.3 Preview UX rules

- preview must be clearly labeled as preview
- preview must not activate the source feed
- preview must not create canonical artifacts
- preview should increase confidence, not overwhelm

---

## 12.4 Good preview outcomes

The admin should be able to answer:
- Is this the right source?
- Does the content shape match expectations?
- Is the source likely too broad or too sensitive?
- Are the governance settings still appropriate?

---

## 13. Step 7 — Review and Validate

## 13.1 Purpose

Provide a final review screen before activation.

### Show summary of:
- connector
- source mapping
- owner
- domain
- sensitivity
- allowed jobs
- ingestion mode
- sync mode
- preview outcome
- current validation status

---

## 13.2 Activation readiness

The review step should show a clear readiness state.

### Example states
- Ready to activate
- Missing required governance fields
- Preview recommended before activation
- Auth issue must be resolved
- Source mapping incomplete

### UX rule
Admins should know exactly why a feed is or is not ready.

---

## 13.3 Validation rules

A feed must not activate unless it has:
- connector/auth OK where required
- source mapping
- owner
- domain
- sensitivity
- allowed jobs defined
- ingestion mode

Sync mode may be implicit for some connectors, but should still be visible if meaningful.

---

## 14. Step 8 — Activate

## 14.1 Purpose

Move the source feed into active operational state.

### Activation should:
- create or finalize the source feed
- set its state to active
- prepare sync behavior
- show next-step actions

---

## 14.2 Post-activation next steps

After activation, the UI should offer:
- run initial sync
- view feed detail
- inspect previewed source
- configure jobs
- return to source feeds list

### Important rule
Activation should feel like the start of governance, not the end of setup.

---

## 15. Source Feed detail after setup

After activation, the Source Feed detail page should show:
- connector type
- source reference
- owner
- domain
- sensitivity
- allowed jobs
- ingestion mode
- sync mode
- status
- health status
- last successful sync
- recent ingestion runs
- raw artifact count
- normalized record count
- preview action if still useful
- pause/resume/reprocess actions where authorized

This is where operational management begins.

---

## 16. Error states and recovery

The setup flow should support practical recovery from common issues.

### 16.1 Auth failure
Show:
- connector auth failed
- reconnect action
- whether source mapping can continue

### 16.2 Invalid source mapping
Show:
- selected source is unavailable or invalid
- choose another source or refresh mapping

### 16.3 Missing governance fields
Show:
- what is missing
- direct path back to the missing step

### 16.4 Preview failure
Show:
- preview could not be generated
- whether activation may still proceed
- recommended next action

### 16.5 Activation failure
Show:
- activation failed
- likely cause
- whether this is recoverable in UI
- link to detail or ops state if already created

---

## 17. UX copy principles

The Source Feed setup flow should use:
- plain product language
- low jargon
- explicit governance language
- calm, operational wording

### Good language patterns
- Assign owner
- Choose domain
- Set sensitivity
- Select allowed jobs
- Preview source
- Ready to activate

### Avoid
- vague integration jargon
- over-technical backend wording
- connector-internal terms unless necessary

---

## 18. Empty and first-run states

For first-time admins, the Source Feeds area should explain:
- what a source feed is
- why governance fields matter
- what happens after activation
- how feeds connect to jobs and retrieval

### Suggested first-run content
A source feed is a governed knowledge source. Before activation, assign ownership, domain, sensitivity, and allowed jobs so downstream retrieval and AI stay controlled.

---

## 19. Anti-patterns to avoid

Do not:
- collapse the entire flow into one long unmanaged form
- make auth success feel like setup completion
- allow activation with missing governance fields
- hide preview behind advanced settings only
- make allowed jobs feel optional if they are required for governance
- use raw technical error language without practical guidance
- treat Source Feed setup like a casual consumer integration flow

---

## 20. Success criteria

This flow is successful when an admin can:
- connect a source confidently
- understand what source is being used
- assign the right governance settings
- preview the source before activation
- understand readiness clearly
- activate without engineering help in the common path
- manage the feed afterward with confidence

---

## 21. Final UX stance

The Source Feed setup flow should feel like:
- governed onboarding
- not raw technical plumbing
- not a casual plugin install
- not a developer-only operation

A good admin should leave the flow knowing:
- what source was connected
- who owns it
- what domain it belongs to
- how sensitive it is
- what jobs may use it
- whether it is safe and ready to activate
