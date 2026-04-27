import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { ScenariosCatalogClient } from "@/components/control-plane/ScenariosCatalogClient";

export default function ControlPlaneScenariosListPage() {
  return (
    <CpScaffold
      title="Scenarios"
      description="Productized usage patterns: bind roles, sources, jobs, and governance for a recurring context. Browse scenarios + presets and clone presets into editable scenarios."
      screenType="Catalog"
      guidanceSlug="scenarios"
      crossLinks={[
        { href: "/control-plane/scenarios/new", label: "Create scenario" },
        { href: "/control-plane/jobs", label: "Knowledge jobs" },
        { href: "/control-plane/presets", label: "Presets" },
      ]}
    >
      <ScenariosCatalogClient />
    </CpScaffold>
  );
}
