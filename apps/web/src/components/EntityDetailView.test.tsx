import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EntityDetailView, type EntityDetailResponse, type RelatedEntityItem, type EntityEvidenceResponse } from "./EntityDetailView";

describe("EntityDetailView", () => {
  it("shows trust badge and source link when present", () => {
    const detail: EntityDetailResponse = {
      entity: {
        id: "e1",
        type: "ReferenceDocument",
        title: "Doc",
        summary: null,
        body: "hello",
        owner_id: "u1",
        domain_id: "d1",
        sensitivity_level: 0,
        truth_mode: "mirrored_authority",
        lifecycle_state: "draft",
        freshness_status: "unknown",
        canonical_status: "draft",
        approval_status: "none",
        external_ref: "gdrive:abc",
        access_policy_id: null,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
      payload: null,
      provenance: [],
      snapshot_at: new Date().toISOString(),
      source: "google_drive_ingestion",
      open_in_source_url: "https://example.com/doc",
      content_preview: { text: "hello", truncated: false },
      freshness_status: "unknown",
      truth_mode: "mirrored_authority",
      external_ref: "gdrive:abc",
      owner_id: "u1",
      domain_id: "d1",
      sensitivity_level: 0,
      lifecycle_state: "draft",
      canonical_status: "draft",
      approval_status: "none",
      updated_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
      access_policy_id: null,
    };

    const related: RelatedEntityItem[] = [
      {
        entity: {
          id: "e2",
          type: "Insight",
          title: "Related",
          truth_mode: "derived",
          lifecycle_state: "draft",
          freshness_status: "unknown",
          updated_at: new Date().toISOString(),
          domain_id: "d1",
        },
        reason: "linked:related",
      },
    ];

    const evidence: EntityEvidenceResponse = {
      entity_id: "e1",
      can_view_raw: false,
      can_view_normalized: false,
      evidence: [],
    };

    render(<EntityDetailView detail={detail} related={related} evidence={evidence} />);
    expect(screen.getAllByText("Mirrored (authority)").length).toBeGreaterThan(0);
    expect(screen.getByRole("link", { name: "View at source" })).toHaveAttribute("href", "https://example.com/doc");
    expect(screen.getByRole("link", { name: "Related" })).toHaveAttribute("href", "/entities/e2");
  });
});

