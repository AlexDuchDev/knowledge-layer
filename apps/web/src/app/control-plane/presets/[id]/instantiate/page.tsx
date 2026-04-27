import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { PreviewGuidance } from "@/components/guidance/PreviewGuidance";

export default async function InstantiatePresetPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Instantiate preset"
      description={`Flow: create working copy from preset ${id}, then land in editor (scaffold).`}
      screenType="Builder"
      guidanceSlug="presets"
      crossLinks={[
        { href: `/control-plane/presets/${encodeURIComponent(id)}`, label: "Preset preview" },
        { href: "/control-plane/roles/new", label: "Role editor" },
        { href: "/control-plane/scenarios/new", label: "Scenario editor" },
        { href: "/control-plane/jobs/new", label: "Job editor" },
      ]}
    >
      <PreviewGuidance kind="preset" />
    </CpScaffold>
  );
}
