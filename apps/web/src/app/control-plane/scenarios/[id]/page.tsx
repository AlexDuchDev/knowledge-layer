import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { ScenarioDetailClient } from "@/components/control-plane/ScenarioDetailClient";

export default async function EditScenarioPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Scenario detail"
      description={`Configuration for scenario ${id}. The scenario itself is configuration — its outputs surface in /knowledge or /digests via the bound jobs.`}
      screenType="Builder"
      guidanceSlug="scenarios"
      crossLinks={[
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}/preview`, label: "Preview" },
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}/bindings/roles`, label: "Role bindings" },
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}/bindings/sources`, label: "Source bindings" },
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}/bindings/jobs`, label: "Job bindings" },
        { href: `/control-plane/jobs`, label: "Bound jobs" },
      ]}
    >
      <ScenarioDetailClient id={id} />
    </CpScaffold>
  );
}
