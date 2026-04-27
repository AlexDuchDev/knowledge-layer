import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { EmptyStateGuidance } from "@/components/guidance/EmptyStateGuidance";

export default function ControlPlaneJobsListPage() {
  return (
    <CpScaffold
      title="Knowledge Jobs"
      description="Configured jobs (catalog). Job runs are operational history, not the same object as the definition."
      screenType="Catalog"
      guidanceSlug="jobs"
      crossLinks={[
        { href: "/control-plane/jobs/new", label: "Create from preset" },
        { href: "/control-plane/jobs/new/custom", label: "Custom job" },
        { href: "/control-plane/scenarios", label: "Scenarios" },
        { href: "/control-plane/sources", label: "Source feeds" },
      ]}
    >
      <EmptyStateGuidance
        title="Job catalog"
        body="Define automated knowledge work here: triggers, source scope, output policy, and sensitivity. Runs are executed by the job worker and appear under each job’s runs view."
        primaryAction={{ href: "/control-plane/jobs/new", label: "Create from preset" }}
        secondaryActions={[{ href: "/control-plane/jobs/new/custom", label: "Custom job" }]}
      />
    </CpScaffold>
  );
}
