import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CatalogPageTemplate } from "./CatalogPageTemplate";

describe("CatalogPageTemplate", () => {
  it("renders title, catalog chip, and children", () => {
    render(
      <CatalogPageTemplate title="Test catalog" description="Desc">
        <p>Child content</p>
      </CatalogPageTemplate>,
    );
    expect(screen.getByRole("heading", { level: 1, name: "Test catalog" })).toBeInTheDocument();
    expect(screen.getByText("Catalog")).toBeInTheDocument();
    expect(screen.getByText("Desc")).toBeInTheDocument();
    expect(screen.getByText("Child content")).toBeInTheDocument();
  });
});
