import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { ApprovalsQueueClient } from "@/components/control-plane/GovernanceQueueClients";

export default function ControlPlaneApprovalQueuePage() {
  return (
    <CpScaffold
      title="Approval queue"
      description="Items awaiting publish approval. Requires the publish capability in the granted domain."
      screenType="Operational"
      guidanceSlug="governance"
      crossLinks={[
        { href: "/control-plane/governance/reviews", label: "Reviews" },
        { href: "/control-plane/governance/policy-exceptions", label: "Policy exceptions" },
        { href: "/control-plane/governance", label: "Governance dashboard" },
      ]}
    >
      <ApprovalsQueueClient />
    </CpScaffold>
  );
}
