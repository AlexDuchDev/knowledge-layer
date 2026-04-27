import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { ReviewsQueueClient } from "@/components/control-plane/GovernanceQueueClients";

export default function ControlPlaneReviewQueuePage() {
  return (
    <CpScaffold
      title="Review queue"
      description="Open review tasks awaiting analyst attention. Drill down to the entity, job, or source the review is anchored to."
      screenType="Operational"
      guidanceSlug="governance"
      crossLinks={[
        { href: "/control-plane/governance/approvals", label: "Approvals" },
        { href: "/control-plane/governance/stale", label: "Stale content" },
        { href: "/control-plane/governance", label: "Governance dashboard" },
      ]}
    >
      <ReviewsQueueClient />
    </CpScaffold>
  );
}
