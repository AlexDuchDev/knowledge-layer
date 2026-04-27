import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function EditJobPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Edit job"
      description={`Configured job ${id}. Distinct from a single run in run history.`}
      screenType="Builder"
      guidanceSlug="jobs"
      crossLinks={[
        { href: `/control-plane/jobs/${encodeURIComponent(id)}/preview`, label: "Preview" },
        { href: `/control-plane/jobs/${encodeURIComponent(id)}/test`, label: "Test run" },
        { href: `/control-plane/jobs/${encodeURIComponent(id)}/runs`, label: "Run history" },
        { href: "/control-plane/sources", label: "Source feeds using this job" },
        { href: "/control-plane/scenarios", label: "Scenarios" },
      ]}
    />
  );
}
