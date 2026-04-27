# SEARCH_AND_QA_UX.md

## 1. Purpose

This document defines the user experience for Search and Scoped Q&A in v1 of the Organizational Memory & Knowledge Operations Platform.

It covers:
- the main search experience
- the main Ask / Scoped Q&A experience
- result and answer structures
- trust indicators
- citations and supporting entities
- partial-answer behavior
- weak-evidence behavior
- feedback patterns
- key states and interactions

This is not a visual design system spec.
It is the product UX contract for how users discover and consume trusted knowledge.

---

## 2. UX goals

Search and Q&A should help users:
- find the right knowledge quickly
- understand whether it is trustworthy
- inspect supporting evidence easily
- move from answer to source to related context
- understand when information may be incomplete, stale, or derived
- avoid being misled by polished but weak AI output

The experience should feel:
- fast
- grounded
- explainable
- calm
- enterprise-credible

It should not feel:
- magical but vague
- chatty without evidence
- overloaded with governance internals
- indistinguishable between strong and weak sources

---

## 3. Product stance

Search and Ask are related but distinct.

### Search
Search is the primary discovery surface for:
- exact lookup
- filtered exploration
- browsing within known scopes
- finding artifacts by title, keyword, type, owner, domain, freshness, or status

### Ask
Ask is the primary synthesis surface for:
- natural-language questions
- decision context reconstruction
- process and policy clarification
- cross-source summary within allowed scope

Search helps users find.
Ask helps users understand.

Both must preserve:
- permission-awareness
- trust semantics
- provenance visibility
- supporting evidence

---

## 4. Search UX

## 4.1 Search page structure

The Search page should include:

1. Search input
2. Scope/preset controls
3. Filter bar
4. Results area
5. Optional result grouping or tabs
6. Empty/no-results states

### Recommended page layout

- top search bar
- directly beneath it:
  - common presets
  - current scope summary
- left or top filter controls depending on final UI density
- main results panel
- optional right rail later for related guidance or recent items

---

## 4.2 Search input

The search input should support:
- free-text entry
- known titles
- exact keywords
- question-like search
- quick reruns with edited phrasing

The input should feel:
- prominent
- fast
- clean
- not overloaded with advanced controls

### Placeholder examples
- Search decisions, policies, meetings, and insights
- Find the latest approved onboarding SOP
- Search finance decisions from last quarter

---

## 4.3 Search presets and common scopes

The UI should support simple presets such as:
- All
- Decisions
- Policies
- SOPs
- Meetings
- Insights
- My domain
- Approved only, where appropriate

These should:
- speed up common flows
- not replace filters
- remain clearly visible and reversible

Presets are a UX accelerator, not a permission bypass.

---

## 4.4 Search filters

At minimum, Search should support:
- entity type
- domain
- owner
- truth mode
- lifecycle state
- freshness status
- approval status where relevant

### Filter behavior rules
- filters should be understandable without training
- filter values should match system terminology
- trust-sensitive filters should be especially clear
- filters should not expose restricted values that imply hidden content unless allowed

### Important trust-related filters
- Canonical / Mirrored / Derived
- Approved / In Review / Draft
- Fresh / Review Due / Stale
- Confirmed / Proposed / Superseded for decisions

---

## 4.5 Search results

Each result should feel like a trustworthy artifact card or row, not just a text snippet.

### Each result should show
- title
- entity type
- short snippet or summary
- owner
- domain
- trust mode badge
- workflow or lifecycle badge where relevant
- freshness badge where relevant
- last updated or reviewed time where useful

### Optional secondary metadata
- linked project
- related meeting
- source type marker for mirrored/reference content

### Design rule
Trust should be scannable at the same speed as relevance.

---

## 4.6 Search result trust badges

Results should support badges such as:
- Canonical
- Mirrored
- Derived
- Approved
- In Review
- Draft
- Stale
- Superseded
- Confirmed

Not every result needs every badge.
Only meaningful trust indicators should appear.

### Badge usage rules
- avoid badge overload
- show the most important trust distinctions first
- do not hide critical trust state in secondary detail only

---

## 4.7 Search result actions

Each result should support:
- open detail
- open related content where relevant
- optional “ask about this” later or where implemented
- maybe copy link/share later if allowed

Search results should not try to become mini detail pages.

---

## 4.8 Search result ordering

The user does not need to see ranking internals, but the UX must reflect the system’s trust-aware ranking model.

The visible behavior should usually make it feel like:
- stronger trusted artifacts appear first when appropriate
- stale or superseded content is still visible when relevant, but clearly labeled
- weak derived content does not quietly masquerade as the best answer

---

## 4.9 Search empty state

If no results are found, the page should:
- say so clearly
- suggest filter or query adjustments
- offer to switch to broader search or Ask where appropriate

### Good no-result guidance examples
- No results matched this query in your current scope
- Try removing one or more filters
- Try Ask for a broader synthesis across permitted knowledge

Do not imply the system searched everything if the result was permission-scoped.

---

## 4.10 Search weak-result state

Sometimes the system will return poor or sparse results.

The UI should support a weak-result state such as:
- limited relevant results found
- results are mostly derived or stale
- query may be too broad or too specific

This should be informative, not alarming.

---

## 4.11 Taxonomy explorer alongside Search

Search should not be the only way into the corpus. A **taxonomy explorer** complements keyword and semantic search: users navigate **domain → entity type → optional facets** (e.g. approval, freshness) in a visible tree or panel, then see a scoped list or run search within that slice.

### Goals
- Reduce “search-only” fatigue for users who think in **where** (which domain) and **what** (decision vs policy vs meeting).
- Align with **scope-first retrieval** ([AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md) §5.8): the UI makes the chosen slice explicit before results load.

### Placement (product intent)
- **Search page:** optional left rail or collapsible panel listing domains the user may access and standard type shortcuts (mirroring `/knowledge` browse and presets in §4.3).
- **Knowledge hub (`/knowledge`):** remains the type index; links into Search with **pre-applied filters** should keep the explorer and search bar in sync.

### Rules
- Explorer choices are **filters on permitted scope**, not a grant.
- Switching domain or type in the explorer updates the **scope summary** (§4.3) and the result set.
- Trust badges and filters (§4.4–4.6) still apply; the explorer does not hide lifecycle or truth mode.

---

## 4.12 Guided exploration (“traverse”) from a seed entity

From **entity detail**, users should move outward to **related governed objects** in a bounded, trustworthy way—similar in ergonomics to graph traversal in local memory tools, but **always permission-checked and trust-labeled**.

### Entry points
- **Related** section on entity detail: show linked entities (existing `entity_links` / detail API) with type, title, trust badges, and short relationship hint where available.
- **Explore from here** (optional affordance): open a focused view or panel listing **1-hop (and at most 2-hop where implemented)** neighbors, with each hop evaluated for `view` on the target.

### UX rules
- Each related item is a **first-class entity row** (title, type, domain, trust); not a raw snippet masquerading as truth.
- If the API omits a link because the user lacks access, the UI must **not** leak existence (consistent with [AI_RETRIEVAL_GOVERNANCE.md](./AI_RETRIEVAL_GOVERNANCE.md) §7.3).
- Relation expansion used in Search hits (e.g. `relation_expansion`) should use the same trust language as detail-page related items.

### Anti-patterns
- Unbounded “show everything connected” without depth limits or trust labels.
- Merging narratives across domains into a single summary without citations and scope.

---

## 5. Ask / Scoped Q&A UX

## 5.1 Ask page structure

The Ask page should include:

1. question input
2. optional scope controls
3. answer area
4. trust indicator area
5. citations area
6. supporting entities area
7. feedback controls
8. optional follow-up prompt area

The overall feel should be:
- focused
- calm
- evidence-first
- not chatbot-gimmicky

---

## 5.2 Question input

The question input should support:
- natural-language questions
- follow-up questions
- scope-sensitive questions
- practical work questions

### Example questions
- What did we decide about onboarding ownership?
- What changed in last week’s planning?
- What is the latest approved process for customer escalations?
- What blockers are still open in finance ops?

The input area should not look like a generic consumer chatbot.
It should look like a trusted workplace knowledge tool.

---

## 5.3 Scope controls for Ask

Ask should optionally support:
- current domain
- entity type narrowing
- approved-only mode where useful
- in-entity scoped mode when launched from a detail page

### Scope UX rules
- current scope should be visible
- scope should be reversible
- scope should never imply access outside the user’s allowed permissions
- if the answer is limited by scope, that should be visible in the answer

---

## 5.4 Answer structure

A strong answer should include:
1. concise answer
2. trust/context note if needed
3. citations
4. supporting entities
5. feedback actions

### Recommended answer layout

#### Primary answer
A concise direct answer or summary.

#### Supporting note
Only when needed:
- partial view
- stale evidence included
- answer based on derived artifacts
- approved-only mode active
- insufficient evidence

#### Citations
Inspectable references to supporting artifacts.

#### Supporting entities
Linked governed objects that help the user continue.

---

## 5.5 Trust indicators in Ask

The answer surface should show relevant trust indicators such as:
- Partial View
- Approved Sources
- Derived Evidence
- Stale Source Included
- Canonical Source Used
- Mirrored Source Used

### Usage rules
- show only what helps interpretation
- avoid badge spam
- make trust signals visible without requiring a deep click

---

## 5.6 Partial-answer behavior

When access or evidence limits the answer, the UI should say so clearly.

### Examples
- This answer may be incomplete due to your access scope
- This answer is based on the strongest permitted evidence available
- Strong approved evidence was not found in your current scope

The system should not:
- imply completeness falsely
- fill gaps with overconfident phrasing
- expose restricted object existence unless allowed

---

## 5.7 Weak-evidence behavior

If evidence is weak or ambiguous, Ask should:
- ask a clarifying question
- say evidence is insufficient
- suggest opening likely relevant entities
- or provide a cautious, bounded answer

### Examples
- I found related materials, but not enough strong evidence for a confident answer
- Do you want the latest approved policy only, or all related discussions?
- I found mostly derived summaries, not a confirmed source of truth

This behavior is a strength, not a failure.

---

## 5.8 Approved-only / trusted-answer mode

For process/policy-like questions, the UX may support a stronger-trust answer mode.

In this mode:
- stronger trusted artifacts are preferred
- the answer should say that it is grounded in approved or canonical material where true
- insufficient approved evidence should be surfaced honestly

### UX requirement
If approved-only mode is active, it should be visible.

---

## 5.9 Follow-up questions

Ask should support lightweight follow-up behavior, but only in a controlled way.

### V1-safe follow-up patterns
- refine the prior question
- ask about cited entities
- ask for narrower scope
- ask for more detail from the same answer context

The product should avoid implying unrestricted persistent memory if that is not actually implemented.

---

## 5.10 Ask-in-context

When Ask is launched from an entity detail page, it should clearly indicate the scope.

### Example UX language
- Asking within this Decision
- Asking within this Policy
- Asking within this Meeting and its supporting context

The user should understand that this is not a global company-wide ask.

---

## 6. Citations UX

## 6.1 Citation goals

Citations should help users:
- inspect evidence
- verify the answer
- move into supporting artifacts
- understand the authority level of sources

Citations are not decorative.

---

## 6.2 Citation display rules

Citations should:
- be easy to scan
- link to meaningful inspectable objects
- not default only to low-level chunks if a better artifact exists
- preserve trust mode where helpful

### A citation item should ideally show
- title
- type
- trust badge or status if relevant
- optional short supporting snippet

---

## 6.3 Citation hierarchy

Where possible, citations should prefer:
1. strong canonical artifact
2. approved governed artifact
3. mirrored authoritative source representation
4. derived artifact
5. low-level chunk evidence only when needed

This hierarchy should influence the UX as well as the backend.

---

## 7. Supporting entities UX

Supporting entities are different from citations.

### Citations answer:
What supports this exact answer?

### Supporting entities answer:
What useful governed artifacts should I inspect next?

Supporting entities should show:
- title
- type
- owner/domain where useful
- trust mode
- important state such as Approved, Stale, Confirmed, Superseded

---

## 8. Feedback UX

Ask should support lightweight answer feedback.

### Suggested feedback actions
- Useful
- Not useful
- Likely stale
- Weak citations
- Possibly incorrect
- Incomplete

Feedback should:
- be easy to submit
- not interrupt reading
- tie back to answer trace where applicable

Search may later support similar feedback on results, but Ask is the higher priority.

---

## 9. Entity detail integration

Entity detail pages should connect strongly to Search and Ask.

### Entity detail should support
- open from Search
- open from Ask citations
- Ask about this entity
- open related content
- **Explore from here:** bounded related/traverse UI (§4.12) with trust labels on each link
- inspect provenance
- inspect versions
- link to **raw source evidence** when the user has `view_raw` and policy allows (snippet ≠ canonical; [adr/0006-raw-artifacts-must-be-preserved.md](./adr/0006-raw-artifacts-must-be-preserved.md))

This creates the core user flow:
Search -> Entity -> Ask in context -> Supporting entities -> Related content -> (optional) deeper hop within permission bounds

---

## 10. Search and Ask relationship

Search and Ask should not compete visually.
They should complement each other.

### Good UX pattern
- Search when the user knows what they want
- Ask when the user wants synthesis or interpretation
- easy movement between the two

### Examples
- from Search: “Need a summary instead? Ask”
- from Ask: “Open the source artifacts” / “Search related results”

---

## 11. States and edge cases

## 11.1 Search loading state
Should be calm, readable, and not over-animated.

## 11.2 Ask loading state
Should suggest that the system is grounding the answer, not just “thinking magically.”

### Example UX wording
- Gathering permitted evidence
- Building answer from trusted sources

Keep it short and not theatrical.

## 11.3 Restricted scope state
If relevant, communicate:
- answer may be limited by scope
- no extra restricted details should leak

## 11.4 Stale evidence state
Clearly indicate when stale evidence contributes to an answer or result.

## 11.5 Derived-only state
If the answer relies mostly on derived artifacts, say so where it affects trust interpretation.

---

## 12. Anti-patterns to avoid

Do not:
- make Ask feel like a general-purpose consumer chatbot
- hide trust and freshness in secondary panels only
- show citations that are technically valid but useless to the user
- make Search results look visually identical regardless of trust state
- overuse badges to the point of visual noise
- imply full knowledge coverage when access/evidence is limited
- let weak answers sound polished and final

---

## 13. Accessibility and clarity

Search and Ask should remain:
- readable
- scannable
- keyboard-friendly
- understandable without training
- light on jargon in user-facing copy

Trust concepts can be sophisticated under the hood, but the surface language should stay simple.

---

## 14. Success criteria

This UX is successful when users can:
- find the right artifact quickly
- tell whether it is trustworthy
- ask practical questions and get useful grounded answers
- inspect citations without friction
- move through related knowledge naturally
- understand when an answer is partial, stale, or derived

---

## 15. Final UX stance

Search and Ask should feel like:
- a trusted knowledge experience
- not a generic search page
- not a generic chatbot
- not a governance console in disguise

The best version of this UX makes the product feel:
- fast
- serious
- useful
- grounded
- confident without pretending certainty
