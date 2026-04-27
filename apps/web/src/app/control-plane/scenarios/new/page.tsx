import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default function CreateScenarioPage() {
  return (
    <CpScaffold
      title="Create scenario"
      description="Builder: configure scenario, then bind roles, sources, jobs."
      screenType="Builder"
      guidanceSlug="scenarios"
      crossLinks={[
        { href: "/control-plane/presets", label: "Create from preset" },
        { href: "/control-plane/scenarios", label: "Scenario list" },
      ]}
    />
  );
}
