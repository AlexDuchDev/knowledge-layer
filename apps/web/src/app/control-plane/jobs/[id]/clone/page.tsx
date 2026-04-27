import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function CloneJobPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Clone job"
      description={`From job ${id} (scaffold).`}
      screenType="Builder"
      guidanceSlug="jobs"
      crossLinks={[
        { href: `/control-plane/jobs/${encodeURIComponent(id)}`, label: "Source job" },
        { href: "/control-plane/jobs/new/custom", label: "New custom" },
      ]}
    />
  );
}
