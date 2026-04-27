import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AskPanel } from "./AskPanel";

vi.mock("@/lib/api", () => ({
  apiJson: vi.fn(async (path: string) => {
    if (path === "/domains") {
      return [{ id: "dom-1", name: "Domain 1" }];
    }
    return {
      trace_id: "trace-1",
      answer: "answer text",
      citations: [{ entity_id: "e1", quote: "q" }],
      supporting_entities: [
        {
          entity_id: "e1",
          title: "T",
          domain_id: "d",
          entity_type: "Insight",
          truth_mode: "derived",
          lifecycle_state: "draft",
          freshness_status: "unknown",
        },
      ],
    };
  }),
}));

afterEach(() => cleanup());

describe("AskPanel", () => {
  it("renders ask UI", () => {
    render(<AskPanel entityId="e1" />);
    expect(screen.getByText("Ask about this entity")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ask" })).toBeDisabled();
  });

  it("renders global ask UI", () => {
    const { container } = render(<AskPanel variant="global" />);
    expect(screen.getByText("Ask (governed synthesis)")).toBeInTheDocument();
    const askButtons = container.querySelectorAll('button[type="button"]');
    const submit = [...askButtons].find((b) => b.textContent === "Ask");
    expect(submit).toBeTruthy();
    expect(submit).toBeDisabled();
  });
});

