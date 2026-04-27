import { afterEach, describe, expect, it, vi } from "vitest";
import { docsArticleUrl } from "./docsLinks";

describe("docsArticleUrl", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("returns null when NEXT_PUBLIC_DOCS_BASE_URL is unset", () => {
    vi.stubEnv("NEXT_PUBLIC_DOCS_BASE_URL", "");
    expect(docsArticleUrl("GLOSSARY.md")).toBeNull();
  });

  it("joins base and relative path, trimming slashes", () => {
    vi.stubEnv("NEXT_PUBLIC_DOCS_BASE_URL", "https://example.com/blob/main/docs/");
    expect(docsArticleUrl("/GLOSSARY.md")).toBe("https://example.com/blob/main/docs/GLOSSARY.md");
  });
});
