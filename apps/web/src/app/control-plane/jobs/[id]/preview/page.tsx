import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { PreviewGuidance } from "@/components/guidance/PreviewGuidance";

export default async function JobPreviewPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Job preview"
      description={`Effective configuration preview for job ${id}.`}
      screenType="Preview"
      guidanceSlug="jobs"
      crossLinks={[
        { href: `/control-plane/jobs/${encodeURIComponent(id)}`, label: "Editor" },
        { href: `/control-plane/jobs/${encodeURIComponent(id)}/test`, label: "Test run" },
      ]}
    >
      <PreviewGuidance kind="job" />
    </CpScaffold>
  );
}
