Security & Governance Rules
1. Document Purpose
This document defines the security, governance and control rules for the Organizational Memory / Company Brain platform.
Its purpose is to ensure that:
knowledge is protected according to business boundaries;
access is controlled and auditable;
AI cannot bypass permissions;
ingestion sources are governed;
published knowledge remains trustworthy;
jobs and automation operate within explicit policy;
the system remains manageable as scope grows.

2. Core Principles
2.1 Least Privilege
Every user, connector, job and retrieval flow must operate with the minimum access required.
2.2 Explicit Scope
No source, job, retrieval process or AI action may default to unrestricted scope.
2.3 Policy Before Convenience
If ease of use conflicts with security boundaries, policy wins.
2.4 Inheritance First
Access and governance rules should be inherited from domains, source feeds and jobs whenever possible.
2.5 Overrides Are Exceptional
Object-level exceptions must be rare, explicit and auditable.
2.6 AI Is Not a Trust Authority
AI may summarize, extract and answer only on top of approved and allowed context. AI does not define truth or policy.
2.7 Traceability
Every important mutation, publication, sync and job run must be traceable.
2.8 Separation of Raw and Canonical Knowledge
Raw imported artifacts and approved canonical knowledge objects must remain distinct.

3. Security Boundaries
The platform must enforce boundaries across the following dimensions:
user identity;
team membership;
role assignment;
domain access;
entity type access;
object-level access;
action permissions;
sensitivity level;
source feed scope;
knowledge job scope;
AI retrieval scope.
No component may bypass these boundaries.

4. Identity and Access Rules
4.1 Identity Required
All application access must be authenticated.
4.2 Backend-Enforced Authorization
Authorization must be enforced on the backend for all:
entity reads;
entity writes;
retrieval requests;
AI requests;
connector operations;
job execution;
governance actions.
Frontend visibility is not a security control.
4.3 Role-Based and Domain-Based Model
Access must be resolved using:
user identity;
assigned team(s);
assigned role(s);
domain grants;
optional object-level overrides.
4.4 Explicit Deny Support
The model must support deny rules that override broader inherited permissions.
4.5 Action Separation
The following permissions must remain distinct:
view;
create;
edit;
approve;
archive;
export;
manage_sources;
manage_jobs;
manage_permissions;
manage_policies.

5. Domain Governance Rules
5.1 Domain Ownership
Each domain must have at least one accountable owner.
5.2 Domain Policy Defaults
Each domain must define default values for:
access policy;
sensitivity level;
review expectations;
publication requirements.
5.3 Domain Isolation
Users without domain grants must not see domain materials, domain search results or AI answers derived from that domain.
5.4 Cross-Functional Layer
Shared cross-domain knowledge must be published explicitly into a controlled CrossFunctional or Leadership domain. Raw restricted materials must not become shared by accident.

6. Sensitivity Rules
Supported sensitivity levels:
public_internal
team_restricted
domain_restricted
leadership_restricted
strictly_confidential
6.1 Sensitivity Must Be Explicit
Every canonical entity, source feed and job output must have a sensitivity level.
6.2 Sensitivity Inheritance
If not explicitly overridden, sensitivity is inherited from:
source feed policy;
job output policy;
domain default.
6.3 Restrictive Resolution
If multiple policies apply, the most restrictive effective rule wins unless explicitly overridden by authorized governance action.

7. Source Feed Governance Rules
7.1 Explicit Source Registration
No external source may be ingested unless it has been explicitly registered as a source feed.
7.2 Required Source Feed Metadata
Every source feed must define:
owner;
domain;
knowledge scope;
sensitivity level;
sync mode;
allowed job usage.
7.3 Telegram Rule for v1
Telegram is supported only as a controlled ingestion source. Only explicitly connected Telegram chats may be read. Each connected Telegram chat must have:
owner;
domain mapping;
knowledge scope;
sensitivity level;
allowed jobs.
7.4 Source Feed Scope Limitation
A source feed must not be treated as globally readable just because a connector exists.
7.5 Source Feed Ownership
The feed owner is responsible for:
source legitimacy;
correct domain mapping;
correct sensitivity assignment;
continued relevance of ingestion.

8. Raw Artifact Rules
8.1 Raw Artifact Preservation
Imported source artifacts may be preserved for provenance and reprocessing.
8.2 Raw Does Not Equal Approved
Raw artifacts are not source of truth unless explicitly transformed into canonical approved objects.
8.3 Raw Access Restrictions
Raw artifacts must inherit at least the same sensitivity and domain restrictions as their source feed.
8.4 Raw Artifact Exposure
Raw artifacts must not be exposed broadly by default in UI, search or AI.

9. Canonical Entity Governance Rules
9.1 Canonical Entity Ownership
Every material canonical entity must have an owner user or owner team.
9.2 Lifecycle State Required
Material entity types must support lifecycle states.
9.3 Reviewability
Entities that affect policy, process, compliance, finance, legal obligations or company-wide operations must support review and approval workflows.
9.4 Versioning Required
Any material change to a canonical entity must create a version record.
9.5 Provenance Required
Canonical entities must retain links to:
source feed;
raw artifacts;
producing knowledge job, if any;
approving or publishing actor.

10. Knowledge Job Governance Rules
10.1 Explicit Source Scope
Each knowledge job must define exactly which source feeds it may read. Jobs must never default to full corpus access.
10.2 Explicit Output Policy
Each knowledge job must define:
output domain;
output sensitivity;
publication mode;
review requirement.
10.3 Operator Restrictions
Only explicitly authorized users or roles may run, edit or publish a job.
10.4 Publication Safety
If a job creates outputs from restricted or mixed-domain inputs, publication must default to draft or reviewed mode.
10.5 Cross-Domain Publication
Publishing outputs into broader domains requires explicit configuration. This must not happen by inheritance accident.
10.6 Sanitization Rule
If a job is intended to publish broader summaries from restricted inputs, sanitization rules must be explicit and reviewed.
10.7 Job Auditability
Every job run must log:
trigger source;
operator if manual;
source feeds read;
outputs created;
publication state;
failures or warnings.

11. AI Governance Rules
11.1 AI Must Use Permission-Aware Retrieval
AI must only receive context already filtered by identity, domain, type, object and sensitivity rules.
11.2 No Unrestricted Corpus Access
The LLM layer must not query raw storage, search indexes or databases without scope enforcement.
11.3 Citation Requirement
AI answers must include supporting entities or citations.
11.4 Trace Requirement
Every AI answer must store an answer trace containing:
requesting user;
retrieval scope;
supporting entities/chunks;
model used;
generation timestamp.
11.5 Restricted Content Protection
AI must not answer from restricted content outside the requestor’s allowed scope, even if the model could technically infer it.
11.6 No Autonomous Publishing
AI must not auto-publish critical materials, policies, decisions or high-sensitivity summaries without human review.

12. Search and Retrieval Security Rules
12.1 Search Must Be Permission-Aware
Search results must be filtered before they are returned.
12.2 Search Indexes Are Not Security Boundaries
Even if indexed separately, permissions must still be resolved at query time or precomputed safely.
12.3 No Leakage Through Metadata
Titles, snippets, facets and counts must not leak restricted existence in ways that violate policy.
12.4 Chunk-Level Safety
If chunks are retrieved independently, access checks must still be enforceable at entity or chunk scope.

13. Review and Approval Rules
13.1 Review-Required Types
The following categories should default to review/approval flows:
Policy
SOP
Process
Legal-sensitive summaries
Finance-sensitive summaries
Leadership briefs generated from mixed restricted sources
13.2 Review Ownership
Each reviewable artifact must define:
owner;
reviewer;
due date;
review state.
13.3 Approval Before Publication
When publication mode is reviewed_publish, outputs must remain unpublished until approved.
13.4 Freshness Governance
Entities with operational significance must have review due dates and freshness policies.

14. Audit and Traceability Rules
14.1 Auditable Events
The system must audit at minimum:
access profile changes;
source feed creation/changes;
knowledge job creation/changes;
entity creation;
entity update;
approval actions;
archive actions;
job runs;
sync runs;
AI answer traces.
14.2 Audit Integrity
Audit events must be append-oriented and not casually editable.
14.3 Operational Visibility
Admins/governance roles must be able to inspect:
failed syncs;
failed job runs;
approval bottlenecks;
stale content;
policy exceptions;
suspicious access patterns if implemented later.

15. Export and Data Exposure Rules
15.1 Export Must Be Permission-Protected
Users may export data only if they hold explicit export permission.
15.2 Restricted Exports
Restricted domains should default to disabled export unless explicitly allowed.
15.3 Output Channels
Any external publication or delivery channel added later must inherit output policy and sensitivity restrictions.

16. Administrative Governance Rules
16.1 Manual Policy Configuration in v1
In v1, governance is configured mainly manually at policy level for:
users;
source feeds;
knowledge jobs;
domain defaults.
16.2 Avoid Per-Object Manual Management
Per-object ACL management should be limited to exceptions. The default model must rely on inheritance.
16.3 Authorized Governors
Only authorized admin/governance roles may:
change domain policies;
change sensitivity defaults;
create policy overrides;
change source feed domain mapping;
widen job output scope.

17. Security Defaults for v1
The following defaults should be enforced in the first implementation:
deny by default unless access is granted;
Telegram reads only from explicitly connected chats;
all source feeds require owner, domain and sensitivity;
all jobs require explicit source scope and output policy;
all AI answers require scoped retrieval and citations;
all material entities require owner and domain;
versioning on all material entity updates;
review required for policy/process-sensitive outputs.

18. Governance Operating Model
18.1 Who governs what
Admin: platform-wide configuration and emergency overrides
Domain Owner: domain-level policy stewardship
Source Feed Owner: legitimacy and classification of source feed
Job Owner: correctness and publication safety of job outputs
Reviewer/Approver: publication control for governed artifacts
18.2 Minimum governance cadence
Recommended cadence:
weekly: failed jobs and failed sync review
biweekly: policy exceptions review
monthly: stale content review
monthly: source feed relevance review
monthly: access model drift review

19. Non-Negotiable Rules
The following rules are non-negotiable:
no unrestricted AI access to the corpus;
no hidden bypass of backend authorization;
no ingestion from unregistered Telegram chats;
no cross-domain publication by accident;
no auto-approval of sensitive outputs;
no assumption that frontend hiding equals protection;
no uncontrolled object-level ACL sprawl.

20. Implementation Guidance for Cursor
When implementing this document in Cursor:
keep permission resolution centralized;
apply access checks in service and retrieval layers;
treat source feeds and jobs as policy-bearing objects;
implement inheritance before manual overrides;
ensure audit events are emitted by all critical mutations and runs;
make AI orchestration dependent on resolved scope;
avoid convenience shortcuts that bypass governance.
