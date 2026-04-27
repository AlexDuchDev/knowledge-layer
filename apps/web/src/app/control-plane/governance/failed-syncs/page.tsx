import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { FailedSyncsQueueClient } from "@/components/control-plane/GovernanceQueueClients";

export default function ControlPlaneFailedSyncsPage() {
  return (
    <CpScaffold
      title="Failed syncs"
      description="Recent source feed sync runs that exited with status=failed. Open the feed sync history to see the full attempt log and re-trigger if appropriate."
      screenType="Operational"
      guidanceSlug="governance"
      crossLinks={[
        { href: "/control-plane/sources", label: "Sources" },
        { href: "/control-plane/governance", label: "Governance dashboard" },
      ]}
    >
      <FailedSyncsQueueClient />
    </CpScaffold>
  );
}
