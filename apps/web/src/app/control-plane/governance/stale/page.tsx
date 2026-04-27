import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { StaleContentQueueClient } from "@/components/control-plane/GovernanceQueueClients";

export default function ControlPlaneStaleQueuePage() {
  return (
    <CpScaffold
      title="Stale content"
      description="Entities flagged stale by the freshness scan. Open the entity to refresh, archive, or annotate; link back to the source feed when applicable."
      screenType="Operational"
      guidanceSlug="governance"
      crossLinks={[
        { href: "/control-plane/sources", label: "Sources" },
        { href: "/governance/freshness-risk", label: "Freshness-risk view (product)" },
        { href: "/control-plane/governance", label: "Governance dashboard" },
      ]}
    >
      <StaleContentQueueClient />
    </CpScaffold>
  );
}
