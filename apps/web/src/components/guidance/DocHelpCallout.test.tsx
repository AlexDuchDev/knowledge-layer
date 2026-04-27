import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DocHelpCallout } from "./DocHelpCallout";

describe("DocHelpCallout", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("renders concept title and summary", () => {
    vi.stubEnv("NEXT_PUBLIC_DOCS_BASE_URL", "");
    render(<DocHelpCallout slug="roles" />);
    expect(screen.getByText("What is a Role?")).toBeInTheDocument();
    expect(screen.getByText(/reusable permission template/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /in-app getting started/i })).toHaveAttribute("href", "/help/getting-started");
  });

  it("shows doc link when base URL is set", () => {
    vi.stubEnv("NEXT_PUBLIC_DOCS_BASE_URL", "https://example.com/blob/main/docs");
    render(<DocHelpCallout slug="ask" />);
    const link = screen.getByRole("link", { name: /read full doc in repository/i });
    expect(link).toHaveAttribute("href", "https://example.com/blob/main/docs/SEARCH_AND_QA_UX.md");
  });
});
