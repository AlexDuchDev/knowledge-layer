# ADMIN_UI_V1.md

## 1. Purpose

This document defines the main admin and governance surfaces for v1.

The admin UI is not a side panel.
It is one of the core product surfaces because this product lives or dies by:
- setup discipline
- policy clarity
- source control
- review workflows
- trust visibility
- operational governability

The goal is to define what the v1 admin UI must let teams do without engineering help.

---

## 2. Main admin surfaces in v1

The initial admin/governance surface should include:

1. Users & Access
2. Source Feeds
3. Knowledge Jobs
4. Governance Center
5. Audit & Operations

These should feel like first-class product areas, not hidden settings pages.

Implementation note (pilot):
- A minimal Governance Center is available in `apps/web` at route `/governance` (queues, policy exceptions, owner remediation, answer feedback, source preview).

---

## 3. Users & Access

### 3.1 Purpose

Allow admins and governance operators to manage:
- users
- teams
- roles
- domain grants
- policy overrides
- effective access understanding

### 3.2 Key screens

#### Users list
Shows:
- name
- email
- status
- primary team
- roles summary
- domain access summary

#### User detail
Shows:
- team memberships
- role bindings
- domain grants
- sensitivity cap if relevant
- explicit overrides
- effective access summary
- recent access-related audit events

#### Roles view
Shows:
- role definitions
- mapped actions
- scope notes

#### Domains view
Shows:
- domain owners
- default policy
- default sensitivity
- users with grants

### 3.3 Required actions

- create/update user
- assign team
- assign role
- assign domain grant
- revoke grant
- create override
- expire override
- inspect effective access

### 3.4 Important UX rule

Admins should be able to understand **why** a user has access, not just see that they do.

---

## 4. Source Feeds

### 4.1 Purpose

Allow admins to configure and operate governed ingestion.

### 4.2 Source feeds list

Should show:
- display name
- connector type
- owner
- domain
- sensitivity
- ingestion mode
- sync mode
- status
- health status
- last successful sync
- allowed jobs summary

### 4.3 Source feed detail

Should show:
- governance metadata
- connector mapping
- source URI or source reference
- access policy
- recent ingestion runs
- raw artifact count
- normalized record count
- warnings/errors
- downstream usage summary
- allowed jobs
- pause/resume state

### 4.4 Source feed create/edit flow

Admin must be able to:
1. choose connector type
2. authenticate connector
3. map source
4. assign owner
5. assign domain
6. assign sensitivity
7. assign allowed jobs
8. choose ingestion mode
9. choose sync mode
10. review settings
11. activate source feed

### 4.5 Required actions

- create source feed
- edit source feed
- activate
- pause
- resume
- trigger sync
- trigger reprocessing
- inspect raw artifacts if authorized
- inspect ingestion run history

### 4.6 Important UX rule

A source feed should feel governed before it feels active.

---

## 5. Knowledge Jobs

### 5.1 Purpose

Allow operators and admins to define and manage repeatable governed operations over knowledge.

### 5.2 Jobs list

Should show:
- job name
- job type
- owner
- trigger type
- status
- output mode
- review requirement
- last run
- next run
- source scope summary

### 5.3 Job detail

Should show:
- purpose
- description
- source scope
- allowed operators
- trigger configuration
- output domain
- output sensitivity
- publication mode
- review requirement
- recent runs
- recent outputs
- related review tasks
- failure summary

### 5.4 Job create/edit flow

Operator should be able to:
1. choose template or blank job
2. define purpose
3. choose source scope
4. choose trigger type
5. set output route
6. set output domain and sensitivity
7. choose review requirement
8. define allowed operators
9. save draft
10. activate when ready

### 5.5 Run detail view

Should show:
- trigger
- initiated by
- resolved input scope snapshot
- status
- outputs created
- warnings
- errors
- links to review tasks
- retry or rerun actions if authorized

### 5.6 Important UX rule

Jobs should feel like operational workflows, not prompt templates.

---

## 6. Governance Center

### 6.1 Purpose

Make review, approval, freshness, and policy exceptions visible and operable.

### 6.2 Main sections

#### Review queue
Shows:
- target title
- target type
- owner
- reviewer
- due date
- truth mode
- sensitivity
- status

#### Approval queue
Shows:
- target
- approver role required
- pending state
- domain
- due date

#### Stale content
Shows:
- title
- entity type
- owner
- domain
- freshness status
- review due date
- linked freshness rule

#### Policy exceptions
Shows:
- target
- override type
- reason
- creator
- expiration
- risk signal

#### Missing governance signals
Shows:
- objects missing owner
- objects missing policy source
- objects missing provenance
- high-sensitivity derived artifacts awaiting review

### 6.3 Required actions

- open review target
- approve
- reject
- request changes
- reassign reviewer
- inspect provenance
- inspect citations
- inspect policy source
- resolve policy override
- trigger stale review workflow

### 6.4 Important UX rule

Governance Center should help teams operate trust, not just observe it.

---

## 7. Audit & Operations

### 7.1 Purpose

Give admins visibility into sensitive actions and system health.

### 7.2 Audit events screen

Should support filters for:
- actor
- event type
- target type
- target ID
- time range
- decision outcome

Should show:
- event type
- actor
- target
- timestamp
- summary reason
- trace ref where applicable

### 7.3 Operations screen

Should show:
- connector health
- ingestion failures
- job failures
- queue state
- recent warnings
- stuck runs if any

### 7.4 Important UX rule

Ops visibility should support action, not just status watching.

---

## 8. Review target experience

This is one of the most important detailed screens in the system.

### 8.1 Reviewer should see

- artifact title
- output type
- truth mode
- owner
- domain
- sensitivity
- workflow state
- generated content
- supporting citations
- supporting entities
- provenance summary
- linked raw/source evidence where allowed
- comments or resolution notes

### 8.2 Required actions

- approve
- request changes
- reject
- open supporting entity
- inspect provenance
- inspect source evidence if permitted

### 8.3 Important UX rule

The reviewer should never have to guess what the artifact is based on.

---

## 9. Search and retrieval admin affordances

Even though search is an end-user surface, admin/governance users need additional controls.

### 9.1 Admin-specific search views

Support:
- search by truth mode
- search by lifecycle
- search by freshness
- search by missing owner
- search by policy source
- search by source feed
- search by job output lineage

### 9.2 Useful admin actions

- open entity detail
- open provenance trail
- open linked source feed
- open job run
- open audit event history

---

## 10. Entity detail expectations in admin surfaces

Admin-facing entity detail should show more than end-user detail.

Required sections:
- content
- ownership
- domain
- sensitivity
- truth mode
- lifecycle state
- freshness
- approval status
- policy source
- provenance
- version history
- linked entities
- audit history summary

---

## 11. UX quality bar for v1

The admin UI does not need enterprise-grade polish in v1.
But it must be:
- understandable
- explicit
- trustworthy
- not overly magical
- fast enough to operate
- good at showing why something is in its current state

That matters more than visual flash.

---

## 12. Priority UI build order

Build in this order:

### First
- Users & Access list/detail
- Source Feeds list/create/detail
- basic Governance review queue

### Then
- Knowledge Jobs list/create/detail
- run detail view
- entity detail with provenance and trust metadata

### Then
- stale content view
- policy exceptions view
- audit list
- ops health panels

This follows the first end-to-end slice.

---

## 13. Anti-patterns to avoid

Do not:
- hide critical controls in tiny settings drawers
- make source feed setup feel like a casual integration click
- collapse review into a single “approve” button with no evidence
- make policy inheritance invisible
- bury truth mode and freshness in secondary tabs only
- build a pretty dashboard that cannot explain actual system behavior

---

## 14. Final admin UI stance

The admin UI should make the system feel governable.

A good admin should be able to answer:
- who can access what
- what sources are connected
- what jobs are running
- what outputs need review
- what is stale
- what is derived
- what changed
- what failed
- why a policy or output is in its current state

If the admin UI cannot answer those questions, the system will not feel trustworthy in real use.