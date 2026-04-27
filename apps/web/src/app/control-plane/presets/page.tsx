import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { PresetsCatalogClient } from "@/components/control-plane/PresetsCatalogClient";

export default function PresetsCatalogPage() {
  return (
    <CpScaffold
      title="Presets"
      description="Curated role, scenario, and job templates. Filter by type or category, open to inspect, instantiate to create a governed editable object."
      screenType="Catalog"
      guidanceSlug="presets"
      crossLinks={[
        { href: "/control-plane/roles/new", label: "New role" },
        { href: "/control-plane/scenarios/new", label: "New scenario" },
        { href: "/control-plane/jobs/new", label: "New job from preset" },
      ]}
    >
      <PresetsCatalogClient />
    </CpScaffold>
  );
}
