import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { PreviewGuidance } from "@/components/guidance/PreviewGuidance";

export default async function ScenarioPreviewPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Scenario preview"
      description={`Effective bindings preview for ${id} (scaffold).`}
      screenType="Preview"
      guidanceSlug="scenarios"
      crossLinks={[
        { href: `/control-plane/scenarios/${encodeURIComponent(id)}`, label: "Editor" },
        { href: `/app/ask`, label: "Try in Knowledge app" },
      ]}
    >
      <PreviewGuidance kind="scenario" />
    </CpScaffold>
  );
}
