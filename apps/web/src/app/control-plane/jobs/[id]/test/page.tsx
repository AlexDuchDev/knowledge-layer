import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function JobTestRunPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Job test run"
      description={`Trigger a test run for job ${id} (scaffold).`}
      screenType="Operational"
      guidanceSlug="jobs"
      crossLinks={[
        { href: `/control-plane/jobs/${encodeURIComponent(id)}/runs`, label: "Run history" },
        { href: `/control-plane/jobs/${encodeURIComponent(id)}`, label: "Job definition" },
      ]}
    />
  );
}
