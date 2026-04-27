import { CpScaffold } from "@/components/control-plane/CpScaffold";

export default function ConnectorsListPage() {
  return (
    <CpScaffold
      title="Connectors"
      description="Connector catalog (plugins). Not the same as a source feed instance."
      screenType="Catalog"
      guidanceSlug="sources"
      crossLinks={[
        { href: "/control-plane/sources/feeds/new", label: "New source feed" },
        { href: "/control-plane/sources", label: "Sources hub" },
      ]}
    />
  );
}
