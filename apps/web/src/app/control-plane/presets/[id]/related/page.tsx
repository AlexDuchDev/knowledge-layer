import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function RelatedPresetsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Related presets"
      description={`Bundles and related entries for preset ${id} (scaffold).`}
      screenType="Catalog"
      guidanceSlug="presets"
      crossLinks={[
        { href: `/control-plane/presets/${encodeURIComponent(id)}`, label: "Preset detail" },
        { href: "/control-plane/presets", label: "Full catalog" },
      ]}
    />
  );
}
