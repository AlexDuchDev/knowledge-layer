import { describe, expect, it } from "vitest";
import { AUTHORING_TEMPLATES, templateById } from "./authoringTemplates";

describe("authoringTemplates", () => {
  it("has unique ids and known types", () => {
    const ids = new Set<string>();
    for (const t of AUTHORING_TEMPLATES) {
      expect(t.id.length).toBeGreaterThan(0);
      expect(ids.has(t.id)).toBe(false);
      ids.add(t.id);
      expect(t.type.length).toBeGreaterThan(0);
      expect(t.body).toContain("##");
    }
  });

  it("resolves templateById", () => {
    expect(templateById("policy")?.label).toBe("Policy");
    expect(templateById("missing")).toBeUndefined();
  });
});
