/** Mirrors `navigation` on GET /auth/me (apps/api/internal/httpserver/nav_visibility.go). */
export type NavigationVisibility = {
  has_domain_grant: boolean;
  may_publish: boolean;
  may_approve: boolean;
  may_manage_source_feed: boolean;
  may_run_job: boolean;
};

export type NavItem = {
  href: string;
  label: string;
  /** If omitted, item is shown whenever the user is authenticated (shell mounted). */
  visible?: (n: NavigationVisibility) => boolean;
};

export type NavGroup = { title: string; items: NavItem[] };

export const NAV_GROUPS: NavGroup[] = [
  {
    title: "Start",
    items: [
      { href: "/", label: "Home" },
      { href: "/help/getting-started", label: "Getting started" },
      { href: "/search", label: "Search" },
      { href: "/ask", label: "Ask" },
      { href: "/knowledge", label: "Browse index", visible: (n) => n.has_domain_grant },
      { href: "/decisions", label: "Decisions", visible: (n) => n.has_domain_grant },
      { href: "/policies", label: "Policies", visible: (n) => n.has_domain_grant },
      { href: "/processes", label: "Processes / SOPs", visible: (n) => n.has_domain_grant },
      { href: "/meetings", label: "Meetings", visible: (n) => n.has_domain_grant },
      { href: "/meeting-tasks", label: "Meeting tasks (extracted)", visible: (n) => n.has_domain_grant },
      { href: "/insights", label: "Insights", visible: (n) => n.has_domain_grant },
      { href: "/projects", label: "Projects", visible: (n) => n.has_domain_grant },
      { href: "/hubs", label: "Topic hubs", visible: (n) => n.has_domain_grant },
    ],
  },
  {
    title: "Governance",
    items: [
      { href: "/governance", label: "Overview", visible: (n) => n.may_publish },
      { href: "/reviews", label: "Reviews", visible: (n) => n.may_approve || n.may_publish },
      { href: "/approvals", label: "Approvals", visible: (n) => n.may_publish },
    ],
  },
  {
    title: "Control plane",
    items: [
      { href: "/control-plane/governance", label: "Operator tools", visible: (n) => n.may_publish },
      {
        href: "/control-plane/sources",
        label: "Sources hub",
        visible: (n) => n.has_domain_grant || n.may_manage_source_feed,
      },
      { href: "/control-plane/jobs", label: "Knowledge jobs", visible: (n) => n.has_domain_grant },
    ],
  },
  {
    title: "Advanced",
    items: [
      { href: "/settings", label: "Instance settings", visible: (n) => n.may_publish },
      { href: "/audit", label: "Audit log", visible: (n) => n.may_publish },
      { href: "/ops/answer-diagnostics", label: "Answer diagnostics", visible: (n) => n.may_publish },
      { href: "/ops/search-insights", label: "Search insights", visible: (n) => n.may_publish },
    ],
  },
];

/** Product ↔ Control plane header switcher: operators with publish rights only (see NAV_GROUPS “Operator tools”). */
export function canAccessZoneSwitcher(nav: NavigationVisibility | null): boolean {
  return !!nav?.may_publish;
}

/** Bootstrap / feeds / sync checklist on Home — only people who operate ingestion or steward the instance. */
export function canSeeInstanceSetupChecklist(nav: NavigationVisibility | null): boolean {
  if (!nav) return false;
  return nav.may_publish || nav.may_manage_source_feed;
}

export function filterNavByVisibility(groups: NavGroup[], nav: NavigationVisibility | null): NavGroup[] {
  if (!nav) {
    return groups
      .map((g) => ({
        title: g.title,
        items: g.items.filter((it) => it.visible === undefined),
      }))
      .filter((g) => g.items.length > 0);
  }
  return groups
    .map((g) => ({
      title: g.title,
      items: g.items.filter((it) => (it.visible ? it.visible(nav) : true)),
    }))
    .filter((g) => g.items.length > 0);
}
