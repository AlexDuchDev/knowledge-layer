import { describe, expect, it } from "vitest";
import { REQUIRED_PRODUCT_APP_PATHS } from "./productAppRoutes";

describe("productAppRoutes manifest", () => {
  it("uses /app prefix", () => {
    expect(REQUIRED_PRODUCT_APP_PATHS.every((p) => p.startsWith("/app"))).toBe(true);
    expect(REQUIRED_PRODUCT_APP_PATHS).toContain("/app/governance/stale");
  });
});
