import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default function SourcesHealthPage() {
  return (
    <CpScaffold
      title="Source health"
      description="Aggregated feed/connector health (scaffold)."
      screenType="Operational"
      guidanceSlug="sources"
      crossLinks={[
        { href: "/control-plane/sources", label: "Sources hub" },
        { href: "/control-plane/governance/failed-syncs", label: "Failed syncs" },
      ]}
    />
  );
}
