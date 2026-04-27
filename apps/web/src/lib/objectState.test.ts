import { describe, expect, it } from "vitest";
import {
  governanceSemantics,
  jobSurfaceSemantics,
  operationalSemantics,
  provenanceSemantics,
  sourceSurfaceSemantics,
} from "./objectState";

describe("objectState semantics", () => {
  it("returns labels and non-empty classes for governance", () => {
    const s = governanceSemantics("in_review");
    expect(s.label).toBe("In review");
    expect(s.className).toContain("amber");
  });

  it("maps operational failed state", () => {
    const s = operationalSemantics("failed");
    expect(s.label).toBe("Failed");
    expect(s.className).toContain("red");
  });

  it("distinguishes preset vs instantiated", () => {
    expect(provenanceSemantics("preset").label).toBe("Preset");
    expect(provenanceSemantics("instantiated").label).toBe("Instantiated");
  });

  it("distinguishes job definition vs run", () => {
    expect(jobSurfaceSemantics("configured_job").label).toBe("Configured job");
    expect(jobSurfaceSemantics("job_run").label).toBe("Job run");
  });

  it("distinguishes connector vs source feed", () => {
    expect(sourceSurfaceSemantics("connector").label).toBe("Connector");
    expect(sourceSurfaceSemantics("source_feed").label).toBe("Source feed");
  });
});
