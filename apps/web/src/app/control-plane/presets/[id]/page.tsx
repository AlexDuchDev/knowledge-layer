import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { PresetDetailClient } from "@/components/control-plane/PresetDetailClient";

export default async function PresetDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Preset detail"
      description={`Preview preset ${id} and optionally instantiate it. The instantiated object is governed like any hand-authored role / scenario / job.`}
      screenType="Detail"
      guidanceSlug="presets"
      crossLinks={[
        { href: "/control-plane/presets", label: "Catalog" },
        { href: "/control-plane/jobs/new", label: "Job builder entry" },
      ]}
    >
      <PresetDetailClient id={id} />
    </CpScaffold>
  );
}
