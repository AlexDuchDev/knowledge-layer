import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { FailedJobsQueueClient } from "@/components/control-plane/GovernanceQueueClients";

export default function ControlPlaneFailedJobsPage() {
  return (
    <CpScaffold
      title="Failed jobs"
      description="Recent job runs that exited with status=failed. Sourced from /ops/failed-runs (identity-admin only). Open the run for execution metrics, or the job for triggers and source scope."
      screenType="Operational"
      guidanceSlug="governance"
      crossLinks={[
        { href: "/control-plane/jobs", label: "Job definitions" },
        { href: "/control-plane/jobs/runs", label: "All recent runs" },
        { href: "/control-plane/governance", label: "Governance dashboard" },
      ]}
    >
      <FailedJobsQueueClient />
    </CpScaffold>
  );
}
