import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { EmptyStateGuidance } from "@/components/guidance/EmptyStateGuidance";

export default function ControlPlaneSourcesHubPage() {
  return (
    <CpScaffold
      title="Sources"
      description="Connectors (plugins) vs source feeds (configured instances). Do not merge into generic settings."
      screenType="Catalog"
      guidanceSlug="sources"
      crossLinks={[
        { href: "/control-plane/sources/connectors", label: "Connectors" },
        { href: "/source-feeds?from=cp", label: "Source feed wizard" },
        { href: "/help/getting-started", label: "Getting started" },
        { href: "/control-plane/sources/health", label: "Source health" },
        { href: "/control-plane/jobs", label: "Jobs consuming feeds" },
      ]}
    >
      <EmptyStateGuidance
        title="Start with a connector, then a feed"
        body="Pick a connector type that matches your system, then run the guided wizard to create a source feed (domain, owner, sensitivity, sync). Feeds produce raw artifacts and power ingestion jobs."
        primaryAction={{ href: "/source-feeds?from=cp", label: "Open source feed wizard" }}
        secondaryActions={[
          { href: "/control-plane/sources/connectors", label: "Browse connectors" },
          { href: "/help/getting-started", label: "Step-by-step guide" },
        ]}
      />
    </CpScaffold>
  );
}
