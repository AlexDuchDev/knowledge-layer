import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function ScenarioJobBindingsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Scenario — job bindings"
      description={`Binding screen: knowledge jobs for scenario ${id}.`}
      screenType="Binding"
      guidanceSlug="scenarios"
      crossLinks={[
        { href: "/control-plane/jobs", label: "Jobs catalog" },
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}`, label: "Scenario editor" },
      ]}
    />
  );
}
