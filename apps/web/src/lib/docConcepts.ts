/**
 * In-product guidance slugs → copy + canonical doc file (under docs/).
 * Wording stays short; full definitions live in repo docs.
 */
export type GuidanceSlug =
  | "roles"
  | "scenarios"
  | "jobs"
  | "sources"
  | "presets"
  | "governance"
  | "setup"
  | "ask"
  | "search"
  | "explorer"
  | "digests"
  | "projects"
  | "decisions";

export type ConceptCopy = {
  title: string;
  summary: string;
  docFile: string;
};

export const DOC_CONCEPTS: Record<GuidanceSlug, ConceptCopy> = {
  roles: {
    title: "What is a Role?",
    summary:
      "A role is a reusable permission template. It combines with domain grants and policies to decide what users can view, publish, or administer—without redefining rules for every user.",
    docFile: "GLOSSARY.md",
  },
  scenarios: {
    title: "Scenarios vs jobs",
    summary:
      "A scenario binds roles, source feeds, and jobs into a timed or triggered context. Jobs are executable definitions; scenarios describe when and how those jobs run together.",
    docFile: "CONTROL_PLANE_OVERVIEW.md",
  },
  jobs: {
    title: "Knowledge jobs",
    summary:
      "Jobs are first-class automated tasks with triggers, scope, and governance fields. Only a fixed set of job_type values has a runtime processor today; others fail closed on run. See GET /knowledge-jobs/engine-metadata and repo docs LIMITATIONS.md / OSS_V1_SCOPE.md.",
    docFile: "OSS_V1_SCOPE.md",
  },
  sources: {
    title: "Connector vs source feed",
    summary:
      "A connector is the plugin type (e.g. Slack). A source feed is your governed instance: domain, owner, sensitivity, sync mode, and config. Sync vs normalization depth varies by family—see CONNECTOR_CAPABILITY_MATRIX.md in the repo (not every artifact type is normalized yet).",
    docFile: "CONNECTOR_CAPABILITY_MATRIX.md",
  },
  presets: {
    title: "Presets",
    summary:
      "Presets seed roles, scenarios, or jobs from the catalog. Instantiating creates editable live objects you can tailor—faster than building from scratch.",
    docFile: "preset-catalog.md",
  },
  governance: {
    title: "Review vs approval",
    summary:
      "Review is a human checkpoint on content or outputs. Approval is an explicit authorize step, often stricter, before publication or policy exceptions.",
    docFile: "EXAMPLES.md",
  },
  setup: {
    title: "Setup vs bootstrap",
    summary:
      "If the instance has no domain yet, complete API/bootstrap first. Control-plane setup templates and sessions are partial operator aids—not a guaranteed end-to-end wizard until documented otherwise.",
    docFile: "onboarding-setup-flow.md",
  },
  ask: {
    title: "Ask and citations",
    summary:
      "Ask answers over your permitted corpus and returns citations to entities—not unconstrained chat. Scope follows domain grants and sensitivity.",
    docFile: "SEARCH_AND_QA_UX.md",
  },
  search: {
    title: "Scoped search",
    summary:
      "Search returns entities you may view in granted domains (and matching sensitivity). It is not a search of the whole company unless you have broad grants.",
    docFile: "SEARCH_AND_QA_UX.md",
  },
  explorer: {
    title: "Explorer",
    summary:
      "Browse and open entities and relationships within your grants. Use filters to stay inside the right domain and type.",
    docFile: "USER_FACING_PRODUCT_SURFACE.md",
  },
  digests: {
    title: "Digests",
    summary:
      "Digests are recurring summaries produced by jobs (e.g. weekly) scoped to sources and policies. They land in digest surfaces after successful runs.",
    docFile: "KNOWLEDGE_JOBS.md",
  },
  projects: {
    title: "Project memory",
    summary:
      "Project views group knowledge for a thread of work. Access still follows domain grants and entity policies.",
    docFile: "USER_FACING_PRODUCT_SURFACE.md",
  },
  decisions: {
    title: "Decisions",
    summary:
      "Decisions are canonical records of organizational choices, with lifecycle and governance like other entity types.",
    docFile: "GLOSSARY.md",
  },
};
