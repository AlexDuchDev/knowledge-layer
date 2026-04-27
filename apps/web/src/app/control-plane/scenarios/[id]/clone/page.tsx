import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function CloneScenarioPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Clone scenario"
      description={`From scenario ${id} (scaffold).`}
      screenType="Builder"
      guidanceSlug="scenarios"
      crossLinks={[
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}`, label: "Source" },
        { href: "/control-plane/scenarios/new", label: "New scenario" },
      ]}
    />
  );
}
