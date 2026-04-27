import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { PreviewGuidance } from "@/components/guidance/PreviewGuidance";

export default async function RolePreviewPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Role preview"
      description={`Effective access preview for role ${id} (scaffold).`}
      screenType="Preview"
      guidanceSlug="roles"
      crossLinks={[
        { href: `/control-plane/roles/${encodeURIComponent(id)}`, label: "Back to editor" },
        { href: `/control-plane/roles/${encodeURIComponent(id)}/assignments`, label: "Assignments" },
      ]}
    >
      <PreviewGuidance kind="role" />
    </CpScaffold>
  );
}
