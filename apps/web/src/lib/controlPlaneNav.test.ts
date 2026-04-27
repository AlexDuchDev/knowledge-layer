import { describe, expect, it } from "vitest";
import { CONTROL_PLANE_NAV_GROUPS, filterControlPlaneNavByVisibility } from "./controlPlaneNav";
import type { NavigationVisibility } from "./navigation";

const pub: NavigationVisibility = {
  has_domain_grant: true,
  may_publish: true,
  may_approve: true,
  may_manage_source_feed: true,
  may_run_job: true,
};

describe("controlPlaneNav", () => {
  it("includes required top-level hrefs when visibility allows", () => {
    const flat = filterControlPlaneNavByVisibility(CONTROL_PLANE_NAV_GROUPS, pub).flatMap((g) => g.items.map((i) => i.href));
    expect(flat).toContain("/control-plane/setup");
    expect(flat).toContain("/control-plane/roles");
    expect(flat).toContain("/control-plane/scenarios");
    expect(flat).toContain("/control-plane/jobs");
    expect(flat).toContain("/control-plane/sources");
    expect(flat).toContain("/control-plane/presets");
    expect(flat).toContain("/control-plane/governance");
    expect(flat).toContain("/control-plane/users");
    expect(flat).toContain("/ask");
  });

  it("hides publish-only items when may_publish is false", () => {
    const flat = filterControlPlaneNavByVisibility(CONTROL_PLANE_NAV_GROUPS, {
      has_domain_grant: true,
      may_publish: false,
      may_approve: false,
      may_manage_source_feed: false,
      may_run_job: false,
    }).flatMap((g) => g.items.map((i) => i.href));
    expect(flat).not.toContain("/control-plane/roles");
    expect(flat).toContain("/ask");
  });
});
