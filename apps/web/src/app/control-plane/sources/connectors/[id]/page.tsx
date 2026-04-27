import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default async function ConnectorDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <CpScaffold
      title="Connector detail"
      description={`Connector ${id}. Use feeds to configure ingestion with governance.`}
      screenType="Detail"
      guidanceSlug="sources"
      crossLinks={[
        { href: "/control-plane/sources/feeds/new", label: "Create feed from connector" },
        { href: "/control-plane/sources/connectors", label: "Connector list" },
      ]}
    />
  );
}
