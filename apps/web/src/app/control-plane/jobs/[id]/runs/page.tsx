import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function JobRunHistoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Job run history"
      description={`Operational screen: runs for configured job ${id}.`}
      screenType="Operational"
      guidanceSlug="jobs"
      crossLinks={[
        { href: `/control-plane/jobs/${encodeURIComponent(id)}`, label: "Job definition" },
        { href: "/control-plane/governance/failed-jobs", label: "Failed jobs queue" },
        { href: "/control-plane/jobs/runs/demo", label: "Job run detail (example id)" },
      ]}
    />
  );
}
