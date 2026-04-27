import { describe, expect, it } from "vitest";
import { REQUIRED_CONTROL_PLANE_PATHS } from "./controlPlaneRoutes";

describe("controlPlaneRoutes manifest", () => {
  it("lists minimum paths for IA verification", () => {
    expect(REQUIRED_CONTROL_PLANE_PATHS).toContain("/control-plane/governance");
    expect(REQUIRED_CONTROL_PLANE_PATHS).toContain("/control-plane/roles");
    expect(REQUIRED_CONTROL_PLANE_PATHS).toContain("/control-plane/jobs");
    expect(REQUIRED_CONTROL_PLANE_PATHS.every((p) => p.startsWith("/control-plane/"))).toBe(true);
  });
});
