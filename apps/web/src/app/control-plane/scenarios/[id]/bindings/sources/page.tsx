import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function ScenarioSourceBindingsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Scenario — source bindings"
      description={`Binding screen: source feeds for scenario ${id}.`}
      screenType="Binding"
      guidanceSlug="scenarios"
      crossLinks={[
        { href: "/control-plane/sources", label: "Sources" },
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}`, label: "Scenario editor" },
      ]}
    />
  );
}
