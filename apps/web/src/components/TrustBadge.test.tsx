import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TrustLine } from "./TrustBadge";

describe("TrustLine", () => {
  it("renders placeholders when lifecycle or freshness missing", () => {
    render(<TrustLine truthMode="derived" />);
    expect(screen.getByText(/lifecycle/)).toBeInTheDocument();
    expect(screen.getByText(/freshness/)).toBeInTheDocument();
  });

  it("renders provided lifecycle and freshness", () => {
    render(<TrustLine truthMode="canonical_in_platform" lifecycleState="draft" freshnessStatus="stale" />);
    expect(screen.getByText(/lifecycle draft/)).toBeInTheDocument();
    expect(screen.getByText(/freshness stale/)).toBeInTheDocument();
  });
});
