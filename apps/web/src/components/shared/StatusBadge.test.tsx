import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders governance label", () => {
    render(<StatusBadge variant={{ kind: "governance", state: "stale" }} />);
    expect(screen.getByText("Stale")).toBeInTheDocument();
  });

  it("renders job surface kind", () => {
    render(<StatusBadge variant={{ kind: "job_surface", state: "job_run" }} />);
    expect(screen.getByText("Job run")).toBeInTheDocument();
  });
});
