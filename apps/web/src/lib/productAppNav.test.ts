import { describe, expect, it } from "vitest";
import { filterProductAppNavByVisibility, PRODUCT_APP_NAV_GROUPS } from "./productAppNav";
import type { NavigationVisibility } from "./navigation";

const full: NavigationVisibility = {
  has_domain_grant: true,
  may_publish: true,
  may_approve: true,
  may_manage_source_feed: true,
  may_run_job: true,
};

describe("productAppNav", () => {
  it("includes required paths for full visibility", () => {
    const flat = filterProductAppNavByVisibility(PRODUCT_APP_NAV_GROUPS, full).flatMap((g) => g.items.map((i) => i.href));
    expect(flat).toContain("/ask");
    expect(flat).toContain("/search");
    expect(flat).toContain("/entities");
    expect(flat).toContain("/reviews");
    expect(flat).toContain("/control-plane/governance");
  });

  it("shows ask/search without domain grant but hides explorer", () => {
    const flat = filterProductAppNavByVisibility(PRODUCT_APP_NAV_GROUPS, {
      has_domain_grant: false,
      may_publish: false,
      may_approve: false,
      may_manage_source_feed: false,
      may_run_job: false,
    }).flatMap((g) => g.items.map((i) => i.href));
    expect(flat).toContain("/ask");
    expect(flat).not.toContain("/entities");
  });
});
