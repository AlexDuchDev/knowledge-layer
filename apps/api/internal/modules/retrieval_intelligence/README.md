# Module: retrieval_intelligence

**Purpose:** HTTP layer for **Ask** and related retrieval/synthesis entrypoints wired to QA and governance.

**Flows:** POST ask with scoped domain, return answer + citations + trace id.

**Integrates with:** `internal/retrieval_intelligence`, `internal/qa`, embeddings/search with permission filters.

**Anti-pattern:** Never assemble LLM context from unauthorized entities—enforcement lives in services, not prompts alone.

**Docs:** [docs/AI_RETRIEVAL_GOVERNANCE.md](../../../../../docs/AI_RETRIEVAL_GOVERNANCE.md).
