import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function ScenarioRoleBindingsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Scenario — role bindings"
      description={`Binding screen for scenario ${id}.`}
      screenType="Binding"
      guidanceSlug="scenarios"
      crossLinks={[
        { href: "/control-plane/roles", label: "Roles catalog" },
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}`, label: "Scenario editor" },
      ]}
    />
  );
}
