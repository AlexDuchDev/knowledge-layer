import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ZoneSwitcher } from "./ZoneSwitcher";

describe("ZoneSwitcher", () => {
  afterEach(() => cleanup());
  it("renders Product and Control plane links", () => {
    render(<ZoneSwitcher active="product" />);
    expect(screen.getByRole("link", { name: "Product" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("link", { name: "Control plane" })).toHaveAttribute("href", "/control-plane/governance");
  });

  it("marks control plane as current when active", () => {
    render(<ZoneSwitcher active="control_plane" />);
    expect(screen.getByRole("link", { name: "Control plane" })).toHaveAttribute("aria-current", "page");
  });
});
