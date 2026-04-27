import type { NavigationVisibility } from "@/lib/navigation";

export type ProductAppNavItem = {
  href: string;
  label: string;
  visible?: (n: NavigationVisibility) => boolean;
};

export type ProductAppNavGroup = { title: string; items: ProductAppNavItem[] };

/** End-user product surface — Ask, Find, Explore, digests, governance queues. */
export const PRODUCT_APP_NAV_GROUPS: ProductAppNavGroup[] = [
  {
    title: "Knowledge",
    items: [
      { href: "/ask", label: "Ask" },
      { href: "/search", label: "Search" },
      { href: "/entities", label: "Explorer", visible: (n) => n.has_domain_grant },
      { href: "/projects", label: "Projects", visible: (n) => n.has_domain_grant },
      { href: "/decisions", label: "Decisions", visible: (n) => n.has_domain_grant },
      { href: "/insights", label: "Digests", visible: (n) => n.has_domain_grant },
    ],
  },
  {
    title: "Governance",
    items: [
      { href: "/governance", label: "Overview", visible: (n) => n.may_publish || n.may_approve },
      { href: "/reviews", label: "Reviews", visible: (n) => n.may_approve || n.may_publish },
      { href: "/approvals", label: "Approvals", visible: (n) => n.may_publish },
      { href: "/control-plane/governance/stale", label: "Stale", visible: (n) => n.may_publish },
    ],
  },
  {
    title: "Admin",
    items: [{ href: "/control-plane/governance", label: "Control plane", visible: (n) => n.may_publish }],
  },
];

export function filterProductAppNavByVisibility(
  groups: ProductAppNavGroup[],
  nav: NavigationVisibility | null,
): ProductAppNavGroup[] {
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
