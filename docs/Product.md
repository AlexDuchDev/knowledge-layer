# PRODUCT.md

## Product name

Organizational Memory & Knowledge Operations Platform

Working positioning:
- organizational memory platform
- company brain infrastructure
- knowledge operations engine
- governed retrieval and synthesis layer

---

## 1. Product summary

We are building a governed platform that turns fragmented company knowledge from chats, meetings, documents, and task systems into structured, traceable, permission-aware organizational memory.

The product is designed for companies that need more than document storage or AI chat over files. It provides controlled ingestion, canonical knowledge objects, governed workflows, knowledge jobs, and permission-aware retrieval with AI-assisted synthesis.

The goal is not to create another workspace.
The goal is to create managed knowledge infrastructure.

---

## 2. Problem

In most companies, critical knowledge lives across:
- Slack
- Telegram
- Email
- meeting notes and transcript tools
- Jira / Trello / boards
- Notion / Confluence / Google Docs
- individual people’s heads

This creates recurring problems:
- decision context gets lost
- teams depend on specific people for context
- onboarding is slow
- the same questions repeat
- operational agreements drift
- important knowledge becomes stale or untrusted
- AI over company data becomes risky without governance
- teams cannot quickly reconstruct why something was decided

The core problem is not lack of storage.
The core problem is lack of structured, governed, operational memory.

---

## 3. Why now

Three shifts make this product timely:

1. Companies already have too much fragmented context across chat, meetings, docs, and tools.
2. AI increases demand for retrieval and synthesis, but also increases risk when access and truth are not controlled.
3. Knowledge-heavy organizations now need operational memory, not just documentation.

The opportunity is to build a system that is useful before full enterprise complexity, but can mature into enterprise-leaning governance over time.

---

## 4. What this product is

This product is:
- a governed organizational memory platform
- a system for controlled ingestion from company knowledge sources
- a canonical knowledge layer for decisions, processes, policies, meetings, and insights
- a knowledge operations engine for scheduled, manual, event-driven, and window-based jobs
- a permission-aware retrieval and AI synthesis layer
- an administrative surface for governance, review, approval, lifecycle, and audit

---

## 5. What this product is not

This product is not:
- a generic knowledge base
- a prettier document workspace
- a simple AI chatbot over all company data
- a file archive
- a replacement for Slack, Telegram, Jira, Notion, or Google Docs
- an unrestricted company copilot with broad access by default

---

## 6. Product principle

The core formula:

Systems of record
-> controlled ingestion
-> normalization
-> knowledge core
-> governed retrieval
-> AI-assisted synthesis
-> workflow and governance

The product must preserve control, traceability, and trust at every step.

---

## 7. Ideal customer profile for v1

The strongest early customer profile is:

Companies with 100-1000 employees where:
- communication is fragmented across chats, meetings, docs, and task systems
- context loss has become operationally expensive
- teams are distributed or cross-functional
- onboarding cost is meaningful
- decision traceability matters
- access boundaries matter across domains
- there is enough process maturity to assign owners and reviewers

Best-fit early teams inside those companies:
- operations-heavy teams
- product and engineering organizations
- finance and business operations
- cross-functional program teams
- leadership teams that need decision traceability

Poor fit for v1:
- very small teams with low process complexity
- companies that only want AI search over docs
- organizations without owners, domains, or governance discipline
- customers expecting a fully autonomous AI system with no review model

---

## 8. Primary wedge for v1

### Decision and operational memory for chat-heavy teams

The first clear wedge is not “all company knowledge.”

The first wedge is:
- ingesting governed communication-heavy sources
- extracting decisions, blockers, commitments, risks, and insights
- turning them into governed knowledge objects
- making them searchable and retrievable within permission scope
- generating structured summaries that support real operational work

This gives us a concrete entry point with obvious value:
- less lost context
- better handoffs
- faster status synthesis
- better decision traceability
- safer AI use

---

## 9. v1 target outcome

By the end of v1, a team should be able to:

1. Connect controlled source feeds
2. Assign source owner, domain, sensitivity, and allowed jobs
3. Ingest source content into raw artifacts and normalized structures
4. Run knowledge jobs such as summaries and extractions
5. Review and approve important outputs
6. Store resulting knowledge objects with provenance and version history
7. Ask questions and retrieve only permitted knowledge with citations
8. Audit what was ingested, generated, reviewed, and accessed

If we cannot do these eight things reliably, v1 is incomplete.

---

## 10. Primary users

### 10.1 Admin / governance operator
Responsible for:
- connecting sources
- assigning owners and policies
- controlling jobs and review requirements
- monitoring governance and audit

### 10.2 Domain owner
Responsible for:
- owning a domain’s feeds and knowledge outputs
- reviewing and approving important generated materials
- maintaining freshness and authority

### 10.3 Operator / team lead / PM / ops lead
Responsible for:
- running knowledge jobs
- reviewing summaries and extracted artifacts
- using the platform for planning, coordination, and context reconstruction

### 10.4 Knowledge consumer
Responsible for:
- searching
- retrieving context
- understanding decisions, process, and current state
- using trusted citations and provenance to act with confidence

---

## 11. Core user problems we solve

### Problem A: decision context disappears
Teams remember the outcome but forget:
- why the decision happened
- what alternatives were considered
- who confirmed it
- what source it came from

### Problem B: operational context is expensive to reconstruct
Managers and ICs lose time reconstructing:
- what happened this week
- what changed in planning
- what blockers matter
- which risks are unresolved

### Problem C: AI without governance is not trusted
Users will not trust AI answers when:
- access boundaries are unclear
- source provenance is missing
- approvals are invisible
- stale information looks current
- generated output is mistaken for authority

### Problem D: knowledge decays
Without owners, reviews, freshness checks, and lifecycle rules:
- documents drift
- SOPs become stale
- policies become unreliable
- teams stop trusting the system

---

## 12. Core value proposition

The product creates value in five ways:

### 12.1 Captures fragmented context
It gathers knowledge from controlled sources instead of forcing teams to rewrite everything manually.

### 12.2 Structures knowledge into usable objects
It creates governed entities such as decisions, meetings, SOPs, policies, and insights.

### 12.3 Preserves traceability
Every important object can point back to raw context, origin, owner, review status, and version history.

### 12.4 Enforces access before retrieval
Users and AI only see what they are allowed to see.

### 12.5 Makes AI usable in serious environments
AI becomes useful because it is bounded by retrieval scope, provenance, citations, and review rules.

---

## 13. Core product principles

### 13.1 Knowledge objects matter more than files
The system is built around canonical entities, not folder hierarchies.

### 13.2 Raw context is preserved
Raw source material may be retained for provenance and future interpretation.

### 13.3 AI is not an authority layer
AI can summarize, extract, and synthesize, but not define truth on its own.

### 13.4 Access control happens before retrieval
The system never retrieves outside a user’s allowed scope and filters later.

### 13.5 Policy inheritance is the default
Access and workflow rules should mostly propagate from domains, sources, jobs, and types.

### 13.6 Governance is a core feature
Review, approval, lifecycle, freshness, audit, and source ownership are not add-ons.

---

## 14. Source-of-truth model

For each entity type, the system must clearly classify the object as one of:

### 14.1 Canonical in platform
The platform is the source of truth for the current state.

Examples:
- Policy
- SOP
- Team Handbook
- Role Handbook

### 14.2 Mirrored authority
The source of truth lives in an external system. The platform stores a normalized view and links.

Examples:
- Jira-driven project state
- Trello-derived initiative state
- Google Doc reference mirrors

### 14.3 Derived artifact
The object is created from source materials and requires explicit interpretation or review before being treated as authoritative.

Examples:
- AI-generated weekly digest
- extracted decision candidate
- summary from chat and meetings
- synthesized planning summary

This distinction must appear both in system behavior and user-facing UX.

---

## 15. Trust model

The product must help users answer:
- why am I seeing this answer?
- what sources support it?
- is this authoritative or derived?
- who owns it?
- when was it last reviewed?
- is it complete, or limited by my access scope?

Important trust indicators in the product:
- citations
- provenance
- owner
- review status
- approval status
- freshness status
- canonical vs mirrored vs derived classification
- restricted / partial scope messaging when applicable

Trust is a core product requirement, not a UX enhancement.

---

## 16. v1 scope

### 16.1 Included sources
- Telegram ingestion mode
- Slack
- Email
- Fireflies
- Granola
- Jira
- Trello
- Notion
- Google Drive / Docs

### 16.2 Included knowledge domains
- Decisions
- Projects
- Processes / SOP
- Policies
- Meetings
- Insights

### 16.3 Included system capabilities
- connector framework
- source feed configuration
- knowledge jobs engine
- access model
- canonical entity model
- versioning
- audit trail
- hybrid retrieval
- AI with scoped retrieval and citations
- admin and governance UI

### 16.4 Telegram v1 restriction
Telegram is used only as a controlled ingestion source in v1.

It is not:
- a universal output channel
- a broad conversational bot interface
- an unrestricted assistant entry point

Only explicitly connected Telegram chats with owners, domains, sensitivity rules, and allowed jobs are supported.

---

## 17. Non-goals for v1

We will not aim to do these in v1:
- replace collaboration tools
- support unrestricted AI interaction across all company knowledge
- support highly custom ontology design per customer
- build fine-grained manual ACL management as the default operating model
- fully automate publication of critical materials without review
- solve every enterprise compliance scenario on day one
- support every connector deeply before the foundation is stable

---

## 18. Product bets

The product is based on several key bets:

### Bet 1
Companies will accept a controlled setup flow if the payoff is trusted knowledge retrieval and decision traceability.

### Bet 2
Users care more about trustworthy synthesis with provenance than raw AI convenience.

### Bet 3
A focused wedge around decisions and operational memory creates stronger pull than a broad “AI knowledge platform” story.

### Bet 4
Governance, access inheritance, and review workflows are differentiators, not friction, for the right customers.

### Bet 5
A modular monolith with strong domain boundaries is the right technical shape for early product maturity.

---

## 19. Main risks

### Product risks
- setup is too heavy for initial adoption
- customers do not have clear owners or domain boundaries
- reviewers do not complete review tasks
- generated outputs are not good enough to earn trust
- users still go back to chat and ignore the governed layer
- “company brain” messaging becomes too broad and abstract

### System risks
- ontology becomes too complex too early
- source-of-truth rules remain ambiguous
- AI scope is not tightly enforced
- access rules become too manual
- source feeds and jobs have no clear owners
- connector reliability becomes the bottleneck before core product value is proven

---

## 20. How we reduce risk

- start with a limited canonical entity model
- start with a clear wedge instead of broad platform sprawl
- use policy inheritance by default
- require owners for source feeds and important artifacts
- fail closed on permissions
- force source-of-truth classification
- require review for sensitive or important derived artifacts
- phase connector expansion after foundation is stable

---

## 21. Success metrics

### 21.1 User value metrics
- time to reconstruct decision context
- time to prepare weekly or planning summaries
- reduction in repeated internal questions
- onboarding speed into team/process context
- successful retrieval rate for common operational questions

### 21.2 Product usage metrics
- number of connected governed source feeds
- weekly active users of retrieval and search
- weekly active reviewers / approvers
- number of confirmed decision objects created
- number of knowledge jobs run successfully
- percentage of outputs reviewed and published

### 21.3 Trust and governance metrics
- percentage of AI answers with valid citations
- zero unauthorized retrieval incidents
- percentage of high-value artifacts with owner assigned
- percentage of critical content reviewed before publish
- percentage of stale artifacts identified and routed for review
- audit coverage for sensitive actions

### 21.4 Quality metrics
- extraction precision for decisions, risks, and action items
- duplicate detection quality
- freshness detection quality
- retrieval relevance quality
- answer usefulness rating for scoped Q&A

---

## 22. Positioning

### Primary positioning
A governed organizational memory platform that turns chats, meetings, docs, and operational systems into trusted, permission-aware company knowledge.

### Secondary positioning
Company brain infrastructure with controlled ingestion, governance, and AI retrieval.

### Sharp wedge positioning
From fragmented company conversations to governed decision memory.

---

## 23. Product narrative

A growing company does not fail because it lacks documents.
It fails because context fragments faster than the organization can absorb it.

Important decisions happen in chat.
Clarifications happen in meetings.
Execution details live in boards.
Policies drift in docs.
Institutional memory stays trapped in a few people.

This product creates a governed memory layer above those systems.

It does not replace the systems where work happens.
It makes the knowledge generated by that work retrievable, traceable, reviewable, and usable.

Over time, the company gains:
- less dependency on memory-hoarding individuals
- faster onboarding
- clearer operational continuity
- safer use of AI
- better decision traceability
- more confidence in what is current, approved, and usable

This is not a document product.
It is operational memory infrastructure.

---

## 24. v1 launch standard

v1 is successful only if:
- at least one real team can connect controlled sources end-to-end
- decisions and operational summaries can be created as governed artifacts
- permission-aware retrieval works reliably
- users can inspect provenance and citations
- important generated outputs can be reviewed before publish
- audit trail exists for sensitive operations
- admins can configure sources and jobs without engineering support for every change