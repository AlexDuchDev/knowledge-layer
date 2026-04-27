import { describe, expect, it } from "vitest";
import {
  canAccessZoneSwitcher,
  canSeeInstanceSetupChecklist,
  filterNavByVisibility,
  NAV_GROUPS,
  type NavigationVisibility,
} from "./navigation";

const full: NavigationVisibility = {
  has_domain_grant: true,
  may_publish: true,
  may_approve: true,
  may_manage_source_feed: true,
  may_run_job: true,
};

describe("canAccessZoneSwitcher", () => {
  it("is false when nav is not loaded", () => {
    expect(canAccessZoneSwitcher(null)).toBe(false);
  });

  it("is false for domain grant without publish", () => {
    expect(
      canAccessZoneSwitcher({
        has_domain_grant: true,
        may_publish: false,
        may_approve: false,
        may_manage_source_feed: true,
        may_run_job: false,
      }),
    ).toBe(false);
  });

  it("is true when may_publish", () => {
    expect(
      canAccessZoneSwitcher({
        has_domain_grant: true,
        may_publish: true,
        may_approve: false,
        may_manage_source_feed: false,
        may_run_job: false,
      }),
    ).toBe(true);
  });
});

describe("canSeeInstanceSetupChecklist", () => {
  it("is false when nav is null", () => {
    expect(canSeeInstanceSetupChecklist(null)).toBe(false);
  });

  it("is true for may_manage_source_feed without publish", () => {
    expect(
      canSeeInstanceSetupChecklist({
        has_domain_grant: true,
        may_publish: false,
        may_approve: false,
        may_manage_source_feed: true,
        may_run_job: false,
      }),
    ).toBe(true);
  });

  it("is false for domain grant only", () => {
    expect(
      canSeeInstanceSetupChecklist({
        has_domain_grant: true,
        may_publish: false,
        may_approve: false,
        may_manage_source_feed: false,
        may_run_job: false,
      }),
    ).toBe(false);
  });
});

describe("filterNavByVisibility", () => {
  it("shows only items without requirements when nav is null", () => {
    const g = filterNavByVisibility(NAV_GROUPS, null);
    const flat = g.flatMap((x) => x.items.map((i) => i.href));
    expect(flat).toContain("/");
    expect(flat).toContain("/search");
    expect(flat).toContain("/ask");
    expect(flat).not.toContain("/decisions");
    expect(flat).not.toContain("/control-plane/users");
  });

  it("shows browse routes when user has domain grant", () => {
    const g = filterNavByVisibility(NAV_GROUPS, {
      has_domain_grant: true,
      may_publish: false,
      may_approve: false,
      may_manage_source_feed: false,
      may_run_job: false,
    });
    const flat = g.flatMap((x) => x.items.map((i) => i.href));
    expect(flat).toContain("/decisions");
    expect(flat).not.toContain("/governance");
  });

  it("shows administration when may_publish", () => {
    const g = filterNavByVisibility(NAV_GROUPS, {
      has_domain_grant: true,
      may_publish: true,
      may_approve: false,
      may_manage_source_feed: false,
      may_run_job: false,
    });
    const flat = g.flatMap((x) => x.items.map((i) => i.href));
    expect(flat).toContain("/settings");
    expect(flat).toContain("/governance");
  });

  it("shows reviews when may_approve without publish", () => {
    const g = filterNavByVisibility(NAV_GROUPS, {
      has_domain_grant: true,
      may_publish: false,
      may_approve: true,
      may_manage_source_feed: false,
      may_run_job: false,
    });
    const flat = g.flatMap((x) => x.items.map((i) => i.href));
    expect(flat).toContain("/reviews");
    expect(flat).not.toContain("/approvals");
  });

  it("keeps all groups for full capabilities", () => {
    const g = filterNavByVisibility(NAV_GROUPS, full);
    expect(g.length).toBe(NAV_GROUPS.length);
    const flat = g.flatMap((x) => x.items.map((i) => i.href));
    expect(flat).toContain("/ops/search-insights");
  });
});
