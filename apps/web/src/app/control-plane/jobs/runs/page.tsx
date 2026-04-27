import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { JobRunsListClient } from "@/components/control-plane/JobRunsListClient";

export default function JobRunsListPage() {
  return (
    <CpScaffold
      title="Recent job runs"
      description="All knowledge job runs across the instance, newest first. Filter by status and job type to triage failures or watch a rollout."
      screenType="Operational"
      guidanceSlug="jobs"
      crossLinks={[
        { href: "/control-plane/jobs", label: "Jobs catalog" },
        { href: "/control-plane/governance/failed-jobs", label: "Failed jobs queue" },
      ]}
    >
      <JobRunsListClient />
    </CpScaffold>
  );
}
