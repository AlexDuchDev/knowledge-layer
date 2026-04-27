import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { JobRunDetailClient } from "@/components/jobs/JobRunDetailClient";

export default async function ControlPlaneJobRunDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Job run"
      description="Raw JSON from GET /job-runs/:id for operators."
      screenType="Detail"
      guidanceSlug="jobs"
      crossLinks={[
        { href: "/control-plane/jobs", label: "All jobs" },
        { href: "/control-plane/governance/failed-jobs", label: "Failed jobs queue" },
      ]}
    >
      <JobRunDetailClient runId={id} footerBackHref="/control-plane/jobs" footerBackLabel="Back to jobs" />
    </CpScaffold>
  );
}
