import type { NavigationVisibility } from "@/lib/navigation";

export type ControlPlaneNavItem = {
  href: string;
  label: string;
  visible?: (n: NavigationVisibility) => boolean;
};

export type ControlPlaneNavGroup = { title: string; items: ControlPlaneNavItem[] };

/** Primary control-plane navigation — mental-model sections, not backend packages. */
export const CONTROL_PLANE_NAV_GROUPS: ControlPlaneNavGroup[] = [
  {
    title: "Setup",
    items: [
      { href: "/help/getting-started", label: "Getting started" },
      { href: "/control-plane/setup", label: "Setup", visible: (n) => n.may_publish },
      { href: "/control-plane/setup/templates", label: "Setup templates", visible: (n) => n.may_publish },
    ],
  },
  {
    title: "Configuration",
    items: [
      { href: "/control-plane/roles", label: "Roles & Access", visible: (n) => n.may_publish },
      { href: "/control-plane/scenarios", label: "Scenarios", visible: (n) => n.may_publish },
      { href: "/control-plane/jobs", label: "Knowledge Jobs", visible: (n) => n.has_domain_grant },
      { href: "/control-plane/sources", label: "Sources hub", visible: (n) => n.has_domain_grant },
      { href: "/source-feeds", label: "Connect data (wizard)", visible: (n) => n.may_manage_source_feed },
      { href: "/control-plane/presets", label: "Presets", visible: (n) => n.may_publish },
    ],
  },
  {
    title: "Operations",
    items: [
      { href: "/control-plane/governance", label: "Governance", visible: (n) => n.may_publish },
      { href: "/control-plane/governance/policy-exceptions", label: "Policy exceptions", visible: (n) => n.may_publish },
      { href: "/control-plane/users", label: "Users", visible: (n) => n.may_publish },
    ],
  },
  {
    title: "Back to product",
    items: [
      { href: "/", label: "Home" },
      { href: "/search", label: "Search" },
      { href: "/ask", label: "Ask" },
    ],
  },
];

export function filterControlPlaneNavByVisibility(
  groups: ControlPlaneNavGroup[],
  nav: NavigationVisibility | null,
): ControlPlaneNavGroup[] {
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
