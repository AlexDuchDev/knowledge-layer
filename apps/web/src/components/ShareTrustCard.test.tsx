import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ShareTrustCard } from "./ShareTrustCard";
import type { EntityDetailResponse } from "./EntityDetailView";

const baseDetail = (): EntityDetailResponse => ({
  entity: {
    id: "e1",
    type: "decision",
    title: "Test decision",
    domain_id: "d1",
    sensitivity_level: 1,
    truth_mode: "canonical_in_platform",
    lifecycle_state: "published",
    freshness_status: "current",
    canonical_status: "canonical",
    approval_status: "approved",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-02T00:00:00Z",
  },
  payload: null,
  provenance: [],
  snapshot_at: "2026-01-01T00:00:00Z",
  source: null,
  open_in_source_url: null,
  content_preview: null,
  freshness_status: "current",
  truth_mode: "canonical_in_platform",
  external_ref: null,
  owner_id: null,
  domain_id: "d1",
  sensitivity_level: 1,
  lifecycle_state: "published",
  canonical_status: "canonical",
  approval_status: "approved",
  updated_at: "2026-01-02T00:00:00Z",
  created_at: "2026-01-01T00:00:00Z",
  access_policy_id: null,
});

describe("ShareTrustCard", () => {
  it("shows trust context in preview when expanded", () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });

    const { container } = render(<ShareTrustCard detail={baseDetail()} />);
    fireEvent.click(screen.getByRole("button", { name: /copy link & trust card/i }));

    const pre = container.querySelector("pre");
    expect(pre?.textContent).toContain("Truth:");
    expect(pre?.textContent).toMatch(/canonical_in_platform/);
    expect(pre?.textContent).toMatch(/published/);
    expect(pre?.textContent).toMatch(/approved/);
    expect(pre?.textContent).toMatch(/current/);
  });
});
