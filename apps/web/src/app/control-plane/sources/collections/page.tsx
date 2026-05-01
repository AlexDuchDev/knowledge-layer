import { CpScaffold } from "@/components/control-plane/CpScaffold";
import { CollectionsClient } from "@/components/manual-upload/CollectionsClient";

export default function ManualCollectionsListPage() {
  return (
    <CpScaffold
      title="Collections (manual uploads)"
      description="Folders for files, pasted text, web pages, and YouTube transcripts. Each collection becomes a governed source feed that knowledge jobs can target."
      screenType="Catalog"
      crossLinks={[
        { href: "/control-plane/sources", label: "Sources hub" },
        { href: "/control-plane/sources/connectors", label: "Connectors" },
        { href: "/source-feeds?from=cp", label: "Source feed wizard" },
        { href: "/control-plane/jobs", label: "Knowledge jobs" },
      ]}
    >
      <CollectionsClient />
    </CpScaffold>
  );
}
